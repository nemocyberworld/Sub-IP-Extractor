package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func isLive(ip string, wg *sync.WaitGroup, results chan<- string, liveIPs chan<- string) {
	defer wg.Done()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", ip)
	} else {
		cmd = exec.Command("ping", "-c", "1", ip)
	}

	err := cmd.Run()
	if err == nil {
		results <- fmt.Sprintf("[✔] %s is live", ip)
		liveIPs <- ip
	} else {
		results <- fmt.Sprintf("[✖] %s is not responding", ip)
	}
}

func parseIPRange(ipRange string) ([]string, error) {
	shortRange := regexp.MustCompile(`^(\d+\.\d+\.\d+\.)(\d+)-(\d+)$`)
	fullRange := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)-(\d+\.\d+\.\d+\.\d+)$`)

	if shortRange.MatchString(ipRange) {
		matches := shortRange.FindStringSubmatch(ipRange)
		base := matches[1]
		start, _ := strconv.Atoi(matches[2])
		end, _ := strconv.Atoi(matches[3])
		var ips []string
		for i := start; i <= end; i++ {
			ips = append(ips, fmt.Sprintf("%s%d", base, i))
		}
		return ips, nil
	} else if fullRange.MatchString(ipRange) {
		matches := fullRange.FindStringSubmatch(ipRange)
		startIP := net.ParseIP(matches[1])
		endIP := net.ParseIP(matches[2])
		var ips []string

		for ip := startIP; !ip.Equal(endIP) && ip != nil; ip = nextIP(ip) {
			ips = append(ips, ip.String())
		}
		ips = append(ips, endIP.String())
		return ips, nil
	}

	return nil, fmt.Errorf("invalid range format")
}

func nextIP(ip net.IP) net.IP {
	ip = ip.To4()
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	return ip
}

func loadIPs(input string) ([]string, error) {
	if _, err := os.Stat(input); err == nil {
		var ips []string
		file, err := os.Open(input)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				ips = append(ips, line)
			}
		}
		return ips, nil
	} else if strings.Contains(input, "-") {
		return parseIPRange(input)
	} else {
		return []string{input}, nil
	}
}

func saveLiveIPs(filename string, liveIPs []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, ip := range liveIPs {
		_, err := file.WriteString(ip + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	fmt.Print("Enter IP, IP range, or filename: ")
	var input string
	fmt.Scanln(&input)

	ips, err := loadIPs(input)
	if err != nil {
		fmt.Println("[!] Error:", err)
		return
	}

	fmt.Printf("\n[+] Scanning %d IP(s) with concurrency...\n\n", len(ips))

	var wg sync.WaitGroup
	results := make(chan string, len(ips))
	liveChan := make(chan string, len(ips))

	for _, ip := range ips {
		wg.Add(1)
		go isLive(ip, &wg, results, liveChan)
	}

	wg.Wait()
	close(results)
	close(liveChan)

	// Print all results
	for res := range results {
		fmt.Println(res)
	}

	// Collect and save live IPs
	var liveIPs []string
	for ip := range liveChan {
		liveIPs = append(liveIPs, ip)
	}

	if len(liveIPs) > 0 {
		err := saveLiveIPs("live_ips.txt", liveIPs)
		if err != nil {
			fmt.Println("[!] Failed to save live IPs:", err)
		} else {
			fmt.Printf("\n[✔] %d live IP(s) saved to live_ips.txt\n", len(liveIPs))
		}
	} else {
		fmt.Println("\n[!] No live IPs found.")
	}
}

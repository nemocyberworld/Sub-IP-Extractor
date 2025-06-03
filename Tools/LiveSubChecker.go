package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// ANSI color codes
const (
	GREEN = "\033[92m"
	RED   = "\033[91m"
	RESET = "\033[0m"
)

const (
	TIMEOUT = 3 * time.Second
	THREADS = 20
)

func isLive(subdomain string, client *http.Client) (bool, string) {
	urls := []string{"http://" + subdomain, "https://" + subdomain}

	for _, url := range urls {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode < 400 {
			return true, url
		}
	}
	return false, ""
}

func checkSubdomains(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var subdomains []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			subdomains = append(subdomains, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Printf("\n🔍 Checking %d subdomains...\n\n", len(subdomains))

	// HTTP client with timeout and skip SSL verification (like verify=False)
	client := &http.Client{
		Timeout: TIMEOUT,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	liveCh := make(chan string, len(subdomains))
	jobs := make(chan string, len(subdomains))

	var wg sync.WaitGroup

	// Worker function
	worker := func() {
		defer wg.Done()
		for sub := range jobs {
			live, url := isLive(sub, client)
			if live {
				fmt.Printf("%s[LIVE]  %s → %s%s\n", GREEN, sub, url, RESET)
				liveCh <- sub
			} else {
				fmt.Printf("%s[DEAD]  %s%s\n", RED, sub, RESET)
			}
		}
	}

	// Start workers
	for i := 0; i < THREADS; i++ {
		wg.Add(1)
		go worker()
	}

	// Send jobs
	for _, sub := range subdomains {
		jobs <- sub
	}
	close(jobs)

	wg.Wait()
	close(liveCh)

	var liveSubs []string
	for sub := range liveCh {
		liveSubs = append(liveSubs, sub)
	}

	fmt.Printf("\n✅ Live subdomains found: %d\n", len(liveSubs))

	if len(liveSubs) > 0 {
		fout, err := os.Create("live_subdomains.txt")
		if err != nil {
			return err
		}
		defer fout.Close()

		for _, sub := range liveSubs {
			fmt.Fprintln(fout, sub)
		}
		fmt.Println("💾 Saved to live_subdomains.txt")
	}

	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run live_subdomain_checker.go subdomains.txt")
		os.Exit(1)
	}

	filePath := os.Args[1]
	if err := checkSubdomains(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "[!] Error: %v\n", err)
		os.Exit(1)
	}
}

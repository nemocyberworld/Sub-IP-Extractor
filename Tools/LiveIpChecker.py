import subprocess
import platform
import ipaddress
import re
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from multiprocessing import cpu_count

def is_live(ip):
    param = "-n" if platform.system().lower() == "windows" else "-c"
    try:
        result = subprocess.run(
            ["ping", param, "1", ip],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL
        )
        return ip, result.returncode == 0
    except Exception:
        return ip, False

def parse_ip_range(ip_range):
    match = re.match(r"(\d+\.\d+\.\d+\.)(\d+)-(\d+)$", ip_range)
    if match:
        base, start, end = match.groups()
        return [f"{base}{i}" for i in range(int(start), int(end)+1)]

    match = re.match(r"(\d+\.\d+\.\d+\.\d+)-(\d+\.\d+\.\d+\.\d+)", ip_range)
    if match:
        start_ip = ipaddress.IPv4Address(match.group(1))
        end_ip = ipaddress.IPv4Address(match.group(2))
        # Expand the IP range fully:
        return [str(ip) for ip in ipaddress.summarize_address_range(start_ip, end_ip)][0].hosts()

    raise ValueError("Invalid IP range format")

def load_ips(target):
    if os.path.isfile(target):
        with open(target, "r") as f:
            return [line.strip() for line in f if line.strip()]
    elif "-" in target:
        return parse_ip_range(target)
    else:
        return [target]

def main():
    target = input("Enter IP, IP range, or filename: ").strip()
    try:
        ips = load_ips(target)
    except Exception as e:
        print(f"[!] Error: {e}")
        return

    print(f"\n[+] Scanning {len(ips)} IP(s) using multithreading...\n")

    max_threads = min(100, len(ips), cpu_count() * 5)

    live_ips = []

    with ThreadPoolExecutor(max_workers=max_threads) as executor:
        futures = [executor.submit(is_live, ip) for ip in ips]
        for future in as_completed(futures):
            ip, status = future.result()
            if status:
                print(f"[✔] {ip} is live")
                live_ips.append(ip)
            else:
                print(f"[✖] {ip} is not responding")

    if live_ips:
        with open("live_ips.txt", "w") as f:
            for ip in live_ips:
                f.write(ip + "\n")
        print(f"\n[✔] Saved {len(live_ips)} live IP(s) to live_ips.txt")
    else:
        print("\n[!] No live IPs found.")

if __name__ == "__main__":
    main()

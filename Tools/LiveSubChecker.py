import requests
from concurrent.futures import ThreadPoolExecutor, as_completed

# Colors for output
GREEN = '\033[92m'
RED = '\033[91m'
RESET = '\033[0m'

# Configuration
TIMEOUT = 3  # seconds
THREADS = 20

def is_live(subdomain):
    """Check if a subdomain is live by testing HTTP/HTTPS."""
    urls = [f"http://{subdomain}", f"https://{subdomain}"]
    for url in urls:
        try:
            response = requests.get(url, timeout=TIMEOUT, allow_redirects=True, verify=False)
            if response.status_code < 400:
                return (subdomain, True, url)
        except requests.RequestException:
            continue
    return (subdomain, False, None)

def check_subdomains(file_path):
    """Read subdomains from file and check which are live."""
    with open(file_path, 'r') as f:
        subdomains = [line.strip() for line in f if line.strip()]

    print(f"\n🔍 Checking {len(subdomains)} subdomains...\n")

    live = []

    with ThreadPoolExecutor(max_workers=THREADS) as executor:
        futures = [executor.submit(is_live, sub) for sub in subdomains]
        for future in as_completed(futures):
            subdomain, status, url = future.result()
            if status:
                print(f"{GREEN}[LIVE]  {subdomain} → {url}{RESET}")
                live.append(subdomain)
            else:
                print(f"{RED}[DEAD]  {subdomain}{RESET}")

    print(f"\n✅ Live subdomains found: {len(live)}")
    if live:
        with open('live_subdomains.txt', 'w') as f:
            f.write("\n".join(live))
        print("💾 Saved to live_subdomains.txt")

if __name__ == "__main__":
    import urllib3
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)  # Suppress SSL warnings

    import sys
    if len(sys.argv) != 2:
        print("Usage: python live_subdomain_checker.py subdomains.txt")
        sys.exit(1)

    check_subdomains(sys.argv[1])

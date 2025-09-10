#!/usr/bin/env python3
"""Reboot ORBI router using the web interface API."""
import re
import sys
import requests
from requests.packages import urllib3

# Suppress SSL warnings for self-signed certificates
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Router Configuration Constants
ROUTER_IP = '192.168.10.154'
USERNAME = 'admin'
PASSWORD = 'Home-404_BOM'



class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    BLUE = '\033[94m'
    YELLOW = '\033[93m'
    CYAN = '\033[96m'
    RESET = '\033[0m'
    BOLD = '\033[1m'

    @staticmethod
    def disable():
        attrs = ['GREEN', 'RED', 'BLUE', 'YELLOW', 'CYAN', 'RESET', 'BOLD']
        for attr in attrs:
            setattr(Colors, attr, '')


if not sys.stdout.isatty():
    Colors.disable()


def get_timestamp_from_reboot_page(session) -> str:
    """Get timestamp from the reboot page form action."""
    try:
        response = session.get(
            f'http://{ROUTER_IP}/reboot.htm',
            verify=False,
            timeout=10
        )
        response.raise_for_status()

        match = re.search(r'timestamp=(\d+)"', response.text)
        if match:
            return match.group(1)

        raise ValueError("Could not extract timestamp from reboot page")

    except (requests.exceptions.Timeout, requests.exceptions.ConnectionError, Exception) as e:
        error_msg = {
            requests.exceptions.Timeout: "Connection timeout",
            requests.exceptions.ConnectionError: "Unable to connect"
        }.get(type(e), str(e))
        raise Exception(f"Failed to get timestamp: {error_msg}") from e


def reboot_router(confirm: bool = False) -> bool:
    """Reboot the ORBI router."""
    if not confirm:
        response = input(
            f"Are you sure you want to reboot the router at "
            f"{Colors.CYAN}{ROUTER_IP}{Colors.RESET}? (yes/no): "
        )
        if response.lower() not in ['yes', 'y']:
            print(f"{Colors.YELLOW}Reboot cancelled.{Colors.RESET}")
            return False

    session = requests.Session()
    session.auth = (USERNAME, PASSWORD)

    try:
        print(f"{Colors.BLUE}🔍 Getting timestamp from reboot page...{Colors.RESET}")
        timestamp = get_timestamp_from_reboot_page(session)

        url = (
            f'http://{ROUTER_IP}/apply.cgi?/reboot_waiting.htm '
            f'timestamp={timestamp}'
        )

        headers = {
            'Content-Type': 'application/x-www-form-urlencoded',
            'Referer': f'http://{ROUTER_IP}/reboot.htm'
        }

        data = {
            'submit_flag': 'reboot',
            'yes': 'Yes'
        }

        print(f"{Colors.BLUE}🔄 Initiating router reboot at {ROUTER_IP}...{Colors.RESET}")

        response = session.post(
            url,
            headers=headers,
            data=data,
            verify=False,
            timeout=30,
            allow_redirects=True
        )

        if response.status_code == 200:
            print(f"{Colors.GREEN}✅ Reboot command sent successfully!{Colors.RESET}")
            print(
                f"\n{Colors.BLUE}📍 The router is now rebooting. "
                f"This typically takes 2-3 minutes.{Colors.RESET}"
            )
            print(
                f"{Colors.YELLOW}⚠️  Your internet connection will be "
                f"temporarily unavailable.{Colors.RESET}"
            )
            print(
                f"\n{Colors.CYAN}You can check if the router is back "
                f"online by:{Colors.RESET}"
            )
            print(
                f"  1. Pinging the router: "
                f"{Colors.YELLOW}ping {ROUTER_IP}{Colors.RESET}"
            )
            print(
                f"  2. Running the device list script: "
f"{Colors.YELLOW}python3 netgear_orbi_list_devices.py{Colors.RESET}"
            )
            return True
        else:
            print(
                f"{Colors.RED}❌ Failed to reboot router. "
                f"Status code: {response.status_code}{Colors.RESET}"
            )
            return False

    except (requests.exceptions.Timeout, requests.exceptions.ConnectionError, Exception) as e:
        error_mapping = {
            requests.exceptions.Timeout: (
                "Request timed out. The router may be unresponsive."
            ),
            requests.exceptions.ConnectionError: "Unable to connect to router"
        }
        error_msg = error_mapping.get(type(e), str(e))
        print(f"{Colors.RED}❌ Error: {error_msg}{Colors.RESET}")
        return False


def main():
    """Main function."""
    print(f"\n{Colors.BOLD}🔄 Netgear Orbi Router Reboot Utility{Colors.RESET}")
    print("━" * 50)

    confirm = False
    if len(sys.argv) > 1:
        if sys.argv[1] in ['--force', '-f']:
            confirm = True
            print(f"{Colors.YELLOW}⚡ Force mode: Skipping confirmation prompt{Colors.RESET}")
        elif sys.argv[1] in ['--help', '-h']:
            print(
                f"\n{Colors.BOLD}Usage:{Colors.RESET} "
                f"python3 netgear_orbi_reboot.py [options]"
            )
            print(f"\n{Colors.BOLD}Options:{Colors.RESET}")
            print(f"  {Colors.CYAN}--force, -f{Colors.RESET}    Skip confirmation prompt")
            print(f"  {Colors.CYAN}--help, -h{Colors.RESET}     Show this help message")
            return 0

    success = reboot_router(confirm)

    return 0 if success else 1


if __name__ == '__main__':
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print(f"\n{Colors.YELLOW}Operation cancelled{Colors.RESET}")
        sys.exit(130)

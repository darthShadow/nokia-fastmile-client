#!/usr/bin/env python3
"""List all attached devices on ORBI router."""
import calendar
import json
import sys
import time
from typing import Dict, List, Optional

import requests
from requests.packages import urllib3

# Suppress SSL warnings for self-signed certificates
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Router Configuration Constants
ROUTER_IP = '192.168.10.154'
USERNAME = 'admin'
PASSWORD = 'Home-404_BOM'  # Replace with your actual password



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


def get_devices(max_retries: int = 3) -> Optional[List[Dict]]:
    """Fetch device information from ORBI router."""
    # Create session with authentication
    session = requests.Session()
    session.auth = (USERNAME, PASSWORD)

    # Build URL with timestamp to prevent caching
    url = f"http://{ROUTER_IP}/DEV_device_info.htm?ts={calendar.timegm(time.gmtime())}"

    # Retry logic for network issues
    for attempt in range(max_retries):
        if attempt > 0:
            time.sleep(attempt)  # Progressive delay

        try:
            response = session.get(url, verify=False, timeout=10)
            response.raise_for_status()

            # Parse device information from response
            for line in response.text.split('\n'):
                if line.startswith('device='):
                    devices_json = line.lstrip('device=')
                    return json.loads(devices_json)

        except (requests.exceptions.Timeout, requests.exceptions.ConnectionError, json.JSONDecodeError, Exception) as e:
            error_msg = {
                requests.exceptions.Timeout: "Connection timeout",
                requests.exceptions.ConnectionError: "Unable to connect",
                json.JSONDecodeError: "Invalid response"
            }.get(type(e), str(e))

            if attempt == max_retries - 1:
                print(f"{Colors.RED}❌ Failed to connect after {max_retries} attempts: {error_msg}{Colors.RESET}")
                return None

    return None


def format_device_info(device: Dict) -> str:
    """Format device information for display."""
    name = device.get('name', 'Unknown Device')
    ip = device.get('ip', 'N/A')
    mac = device.get('mac', 'N/A')
    conn_type = device.get('conn_type', 'Unknown')

    # Check connection status
    status = "Connected"
    if 'backhaul_sta' in device:
        backhaul = device['backhaul_sta']
        if backhaul == 'Good':
            status = "Active (Good)"
        elif backhaul == 'Poor':
            status = "Active (Poor)"
        else:
            status = f"Active ({backhaul})"

    return (
        f"{name:<30} IP: {ip:<15} MAC: {mac:<17} "
        f"Type: {conn_type:<10} Status: {status}"
    )


def main():
    """Main function"""
    print(f"\n{Colors.BOLD}🌐 Netgear Orbi Router - Connected Devices{Colors.RESET}")
    print("━" * 50)

    try:
        # Get devices from router
        print(f"{Colors.BLUE}🔍 Fetching device information from {ROUTER_IP}...{Colors.RESET}")
        devices = get_devices()

        if not devices:
            print(f"{Colors.RED}❌ No devices found or unable to connect to router{Colors.RESET}")
            return 1

        # Display device information
        print(f"\n{Colors.GREEN}✅ Found {len(devices)} devices{Colors.RESET}\n")

        # Separate active and inactive devices
        active_devices = []
        inactive_devices = []

        for device in devices:
            if device.get('backhaul_sta') == 'Good':
                active_devices.append(device)
            else:
                inactive_devices.append(device)

        # Display active devices
        if active_devices:
            print(f"{Colors.GREEN}Active Devices ({len(active_devices)}){Colors.RESET}:")
            print(f"{Colors.CYAN}" + "-" * 80 + f"{Colors.RESET}")
            for device in sorted(
                active_devices,
                key=lambda d: d.get('name', '').lower()
            ):
                print(format_device_info(device))

        # Display other devices
        if inactive_devices:
            section_title = f"Other Devices ({len(inactive_devices)})" if active_devices else f"Devices ({len(inactive_devices)})"
            print(f"\n{Colors.YELLOW}{section_title}{Colors.RESET}:")
            print(f"{Colors.CYAN}" + "-" * 80 + f"{Colors.RESET}")
            for device in sorted(
                inactive_devices,
                key=lambda d: d.get('name', '').lower()
            ):
                print(format_device_info(device))

        print(f"\n{Colors.CYAN}" + "━" * 50 + f"{Colors.RESET}")

    except KeyboardInterrupt:
        print(f"\n{Colors.YELLOW}Operation cancelled{Colors.RESET}")
        return 130
    except Exception as e:
        print(f"{Colors.RED}❌ Error: {str(e)}{Colors.RESET}")
        return 1

    return 0


if __name__ == '__main__':
    sys.exit(main())

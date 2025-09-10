#!/usr/bin/env python3
"""
Reboot ORBI router using the web interface API
"""
import re
import sys
import requests
from requests.packages import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

ORBI_IP = '192.168.10.154'
USERNAME = 'admin'
PASSWORD = 'Home-404_BOM'


def get_timestamp_from_reboot_page(session) -> str:
    """Get timestamp from the reboot page form action"""
    response = session.get(
        f'http://{ORBI_IP}/reboot.htm',
        verify=False,
        timeout=10
    )
    
    match = re.search(r'timestamp=(\d+)"', response.text)
    if match:
        return match.group(1)
    
    raise Exception("Could not extract timestamp from reboot page")


def reboot_router(confirm: bool = False) -> bool:
    """Reboot the ORBI router"""
    if not confirm:
        response = input("Are you sure you want to reboot the router? (yes/no): ")
        if response.lower() not in ['yes', 'y']:
            print("Reboot cancelled.")
            return False
    
    session = requests.Session()
    session.auth = (USERNAME, PASSWORD)
    
    try:
        timestamp = get_timestamp_from_reboot_page(session)
        
        url = f'http://{ORBI_IP}/apply.cgi?/reboot_waiting.htm timestamp={timestamp}'
        
        headers = {
            'Content-Type': 'application/x-www-form-urlencoded',
            'Referer': f'http://{ORBI_IP}/reboot.htm'
        }
        
        data = {
            'submit_flag': 'reboot',
            'yes': 'Yes'
        }
        
        print(f"\nInitiating router reboot at {ORBI_IP}...")
        
        response = session.post(
            url,
            headers=headers,
            data=data,
            verify=False,
            timeout=30,
            allow_redirects=True
        )
        
        if response.status_code == 200:
            print("✓ Reboot command sent successfully!")
            print("\nThe router is now rebooting. This typically takes 2-3 minutes.")
            print("Your internet connection will be temporarily unavailable.")
            print("\nYou can check if the router is back online by:")
            print(f"1. Pinging the router: ping {ORBI_IP}")
            print(f"2. Running the device list script: python3 list_devices.py")
            return True
        else:
            print(f"✗ Failed to reboot router. Status code: {response.status_code}")
            return False
            
    except requests.exceptions.Timeout:
        print("✗ Request timed out. The router may be unresponsive.")
        return False
    except Exception as e:
        print(f"✗ Error: {e}")
        return False


def main():
    """Main function"""
    print("ORBI Router Reboot Utility")
    print("=" * 50)
    
    confirm = False
    if len(sys.argv) > 1:
        if sys.argv[1] in ['--force', '-f']:
            confirm = True
            print("Force mode: Skipping confirmation prompt")
        elif sys.argv[1] in ['--help', '-h']:
            print("\nUsage: python3 reboot_orbi.py [options]")
            print("\nOptions:")
            print("  --force, -f    Skip confirmation prompt")
            print("  --help, -h     Show this help message")
            return 0
    
    success = reboot_router(confirm)
    
    return 0 if success else 1


if __name__ == '__main__':
    exit(main())
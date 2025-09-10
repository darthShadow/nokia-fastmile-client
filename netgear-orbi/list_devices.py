#!/usr/bin/env python3
"""
List all attached devices on ORBI router
"""
import calendar
import json
import time
from typing import Dict, List, Optional

import requests
from requests.packages import urllib3

# Suppress SSL warnings for self-signed certificates
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Hardcoded configuration
ORBI_URL = 'http://192.168.10.154/DEV_device_info.htm'
USERNAME = 'admin'
PASSWORD = 'Home-404_BOM'  # Replace with your actual password


def get_devices(max_retries: int = 3) -> Optional[List[Dict]]:
    """Fetch device information from ORBI router"""
    # Create session with authentication
    session = requests.Session()
    session.auth = (USERNAME, PASSWORD)
    
    # Build URL with timestamp to prevent caching
    url = f"{ORBI_URL}?ts={calendar.timegm(time.gmtime())}"
    
    # Retry logic for network issues
    for attempt in range(max_retries):
        try:
            response = session.get(url, verify=False, timeout=10)
            response.raise_for_status()
            
            # Parse device information from response
            for line in response.text.split('\n'):
                if line.startswith('device='):
                    devices_json = line.lstrip('device=')
                    return json.loads(devices_json)
                    
        except requests.exceptions.RequestException as e:
            if attempt == max_retries - 1:
                print(f"Failed to connect after {max_retries} attempts: {e}")
                return None
            time.sleep(1)  # Wait before retry
            
    return None


def format_device_info(device: Dict) -> str:
    """Format device information for display"""
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
    
    return f"{name:<30} IP: {ip:<15} MAC: {mac:<17} Type: {conn_type:<10} Status: {status}"


def main():
    """Main function"""
    print("ORBI Router - Attached Devices\n")
    print("=" * 80)
    
    try:
        # Get devices from router
        devices = get_devices()
        
        if not devices:
            print("No devices found or unable to connect to router")
            return
            
        # Display device information
        print(f"\nTotal devices: {len(devices)}\n")
        
        # Separate active and inactive devices
        active_devices = []
        inactive_devices = []
        
        for device in devices:
            if 'backhaul_sta' in device and device['backhaul_sta'] == 'Good':
                active_devices.append(device)
            else:
                inactive_devices.append(device)
                
        # Display active devices
        if active_devices:
            print("Active Devices:")
            print("-" * 80)
            for device in sorted(active_devices, key=lambda d: d.get('name', '').lower()):
                print(format_device_info(device))
                
        # Display other devices
        if inactive_devices:
            print(f"\n{'Other Devices:' if active_devices else 'Devices:'}")
            print("-" * 80)
            for device in sorted(inactive_devices, key=lambda d: d.get('name', '').lower()):
                print(format_device_info(device))
                
        print("\n" + "=" * 80)
        
    except Exception as e:
        print(f"Error: {e}")
        return 1
        
    return 0


if __name__ == '__main__':
    exit(main())

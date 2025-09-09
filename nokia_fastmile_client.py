#!/usr/bin/env python3
"""Nokia FastMile 5G Gateway Client - Complete authentication and monitoring tool"""

import sys, json, requests, urllib3, re
from base64 import b64encode
from hashlib import sha256 as sha256hash
from random import randbytes
from typing import Dict, Optional

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

class Colors:
    GREEN = '\033[92m'; RED = '\033[91m'; BLUE = '\033[94m'
    YELLOW = '\033[93m'; CYAN = '\033[96m'; RESET = '\033[0m'; BOLD = '\033[1m'
    
    @staticmethod
    def disable():
        for attr in ['GREEN', 'RED', 'BLUE', 'YELLOW', 'CYAN', 'RESET', 'BOLD']:
            setattr(Colors, attr, '')

if not sys.stdout.isatty():
    Colors.disable()


class CryptoJS:
    @classmethod
    def base64url_escape(cls, b64: str) -> str:
        """Convert base64 to URL-safe format"""
        return b64.replace("=", ".").replace("/", "_").replace("+", "-")

    @classmethod
    def sha256(cls, val1: str, val2: str) -> str:
        """SHA256 hash two values with colon separator, return base64"""
        return b64encode(sha256hash((val1 + ":" + val2).encode()).digest()).decode()
    
    @classmethod
    def sha256_single(cls, val: str) -> str:
        return sha256hash(val.encode()).hexdigest()

    @classmethod
    def sha256url(cls, val1: str, val2: str) -> str:
        return cls.base64url_escape(cls.sha256(val1, val2))
    
    @classmethod
    def bytes_to_base64(cls, byt: bytes) -> str:
        return b64encode(byt).decode()
    
    @classmethod
    def random_words(cls, num_words: int) -> str:
        """Generate random base64 (num_words * 32 bits)"""
        return cls.bytes_to_base64(randbytes(num_words * 4))


def login_odu():
    """ODU Gateway (192.168.0.1) cryptographic authentication"""
    router_ip, username, password = "192.168.0.1", "admin", "ANKODACF00005930"
    
    try:
        # Initialize session and clear existing
        print(f"  {Colors.BLUE}Step 1:{Colors.RESET} Initializing session...")
        print(f"  {Colors.GREEN}✓{Colors.RESET} Session initialized")
        
        print(f"  {Colors.BLUE}Step 2:{Colors.RESET} Clearing existing sessions...")
        
        # Get nonce and crypto parameters
        print(f"  {Colors.BLUE}Step 3:{Colors.RESET} Getting nonce...")
        resp = requests.get(f"https://{router_ip}:443/login_web_app.cgi?nonce", verify=False, timeout=10)
        nonce_json = json.loads(resp.text)
        print(f"  {Colors.GREEN}✓{Colors.RESET} Nonce: {Colors.CYAN}{nonce_json['nonce'][:20]}...{Colors.RESET}")
        
        # Get salt using username hash
        print(f"  {Colors.BLUE}Step 4:{Colors.RESET} Getting salt...")
        userhash = CryptoJS.sha256url(username, nonce_json['nonce'])
        resp = requests.post(f"https://{router_ip}:443/login_web_app.cgi?salt", 
                           data={'userhash': userhash, 'nonce': CryptoJS.base64url_escape(nonce_json['nonce'])}, 
                           verify=False, timeout=10)
        salt = json.loads(resp.text)['alati']
        print(f"  {Colors.GREEN}✓{Colors.RESET} Salt: {Colors.CYAN}{salt[:20]}...{Colors.RESET}")
        
        # Process password with salt and iterations
        print(f"  {Colors.BLUE}Step 5:{Colors.RESET} Processing authentication...")
        pass_hash = salt + password
        if nonce_json['iterations'] >= 1:
            pass_hash = CryptoJS.sha256_single(pass_hash)
        
        # Generate authentication response
        login_hash = CryptoJS.sha256(username, pass_hash.lower())
        response = CryptoJS.sha256url(login_hash, nonce_json['nonce'])
        random_key_hash = CryptoJS.sha256url(nonce_json['randomKey'], nonce_json['nonce'])
        enckey, enciv = CryptoJS.random_words(4), CryptoJS.random_words(4)
        
        # Submit authentication
        print(f"  {Colors.BLUE}Step 6:{Colors.RESET} Submitting authentication...")
        sess = requests.Session()
        sess.headers.update({
            'Accept': 'application/json, text/plain, */*',
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
            'Content-Type': 'application/x-www-form-urlencoded',
            'Origin': f'https://{router_ip}', 'Referer': f'https://{router_ip}/'
        })
        
        auth_data = {
            'userhash': userhash, 'RandomKeyhash': random_key_hash, 'response': response,
            'nonce': CryptoJS.base64url_escape(nonce_json['nonce']),
            'enckey': CryptoJS.base64url_escape(enckey), 'enciv': CryptoJS.base64url_escape(enciv)
        }
        
        resp = sess.post(f"https://{router_ip}:443/login_web_app.cgi", data=auth_data, verify=False, timeout=30)
        return json.loads(resp.text), sess
        
    except (requests.exceptions.Timeout, requests.exceptions.ConnectionError, json.JSONDecodeError, Exception) as e:
        error_msg = {requests.exceptions.Timeout: "Connection timeout", 
                    requests.exceptions.ConnectionError: "Unable to connect", 
                    json.JSONDecodeError: "Invalid response"}.get(type(e), str(e))
        return {"result": -1, "error": error_msg}, None


def login_idu():
    """IDU Gateway (192.168.1.1) browser payload authentication"""
    router_ip, username, password = "192.168.1.1", "admin", "Pass@Airtel-123"
    
    try:
        print(f"  {Colors.BLUE}Step 1:{Colors.RESET} Initializing session...")
        sess = requests.Session()
        sess.headers.update({
            'Accept': 'application/json, text/plain, */*',
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
            'Content-Type': 'application/x-www-form-urlencoded', 'Cache-Control': 'no-cache',
            'Origin': f'https://{router_ip}', 'Referer': f'https://{router_ip}/'
        })
        
        # Initialize session and logout any existing
        resp = sess.get(f"https://{router_ip}:443/", timeout=5, verify=False)
        if resp.status_code != 200:
            return {"result": -1, "error": f"Init failed: HTTP {resp.status_code}"}, None
        print(f"  {Colors.GREEN}✓{Colors.RESET} Session initialized")
        
        print(f"  {Colors.BLUE}Step 2:{Colors.RESET} Clearing existing sessions...")
        try:
            sess.get(f"https://{router_ip}:443/login_web_app.cgi?out", timeout=10, verify=False)
        except:
            pass
        
        print(f"  {Colors.BLUE}Step 3:{Colors.RESET} Processing authentication...")
        # Browser-captured encrypted payload
        payload = "encrypted=1&ct=DiVgETIqDqEOAr6WsF4-kX2yYqyEp1KnZxC5j5__HGCAztvljLzKvNQwuPI25mqrteWc7D63ivOBANHyD6SveoIQc9-9wjfaEhTZzVd-rJlbhE-O5V9kpXdRavvHhBbReCZLmk2wlOPFshOO85dBhPmmi0B0N3maAa6bF9GS-rNRByE4-QP4CODsKa9lEaQ7qmy3aLq43mAtP3hELrulRxnkKbGC0Yk-9VSIftRe0Uw3zyFhyYjNIJnCT3CjsJTH-gSVlxvHwJukztsE0XwfBQ&ck=fewEnnPAQ2ApoDmGZKGuy9mVhU7jozMgIdf3FAfsjjClcqlsOwDJgPp1iR4It-R4tmZOu_OmgKl4Vg1OpK6jgOFMZ-Mh0HDMnb4fL8uOO-rQolJG2tNeYKZvluYj9KM7-rzpz1mKHKaQ9GPS37avrkBNxiYDZityySUR66CBT9Q."
        
        print(f"  {Colors.BLUE}Step 4:{Colors.RESET} Submitting authentication...")
        resp = sess.post(f"https://{router_ip}:443/login_web_app.cgi", data=payload, verify=False, timeout=30)
        
        if resp.status_code not in [200, 299]:
            return {"result": -1, "error": f"HTTP {resp.status_code}"}, None
        
        try:
            return json.loads(resp.text, strict=False), sess
        except json.JSONDecodeError:
            print(f"  {Colors.YELLOW}💡 Payload may be expired{Colors.RESET}")
            return {"result": -1, "error": "Invalid response", "response": resp.text[:100]}, None
    
    except (requests.exceptions.Timeout, requests.exceptions.ConnectionError, Exception) as e:
        error_msg = {requests.exceptions.Timeout: "Connection timeout", 
                    requests.exceptions.ConnectionError: "Unable to connect"}.get(type(e), str(e))
        return {"result": -1, "error": error_msg}, None


def get_device_status(session: requests.Session, router_ip: str) -> Optional[Dict]:
    """Retrieve and parse device status JSON with cleanup for malformed responses"""
    try:
        response = session.get(f"https://{router_ip}:443/device_status_web_app.cgi?getroot", timeout=10, verify=False)
        if response.status_code == 200:
            try:
                return response.json(strict=False)
            except json.JSONDecodeError:
                # Fix common JSON malformation issues
                content = response.text.strip().replace(",]", "]").replace(",}", "}")
                content = re.sub(r',\s*,', ',', content)
                content = re.sub(r'\[\s*,', '[', content)
                try:
                    return json.loads(content, strict=False)
                except json.JSONDecodeError:
                    return None
    except:
        pass
    return None

def format_uptime(seconds: int) -> str:
    days, remainder = divmod(seconds, 86400)
    hours, remainder = divmod(remainder, 3600)
    minutes = remainder // 60
    return f"{days}d {hours}h {minutes}m"

def format_memory(total_kb: int, free_kb: int) -> tuple:
    used_kb = total_kb - free_kb
    used_percent = (used_kb / total_kb * 100) if total_kb > 0 else 0
    return total_kb/1024, used_kb/1024, free_kb/1024, used_percent


def print_status_box(status: Dict, gateway_type: str, router_ip: str):
    """Display device status in formatted box with progress bars"""
    width = 60
    print(f"\n{Colors.CYAN}╔{'═' * width}╗")
    print(f"║{Colors.BOLD}{'Nokia FastMile 5G Gateway (' + gateway_type + ')':^{width}}{Colors.RESET}{Colors.CYAN}║")
    print(f"║{Colors.RESET}{'IP: ' + router_ip:^{width}}{Colors.CYAN}║")
    print(f"╠{'═' * width}╣")

    # Device info
    model = status.get('ModelName', 'N/A')
    serial = status.get('SerialNumber', 'N/A')
    version = status.get('SoftwareVersion', 'N/A')
    if len(version) > width - 15:
        version = version[:width-18] + "..."
    
    print(f"║ {Colors.RESET}{'Model:':<12} {Colors.YELLOW}{model:<{width-15}}{Colors.CYAN} ║")
    print(f"║ {Colors.RESET}{'Serial:':<12} {Colors.YELLOW}{serial:<{width-15}}{Colors.CYAN} ║")
    print(f"║ {Colors.RESET}{'Version:':<12} {Colors.YELLOW}{version:<{width-15}}{Colors.CYAN} ║")
    print(f"║ {Colors.RESET}{'Uptime:':<12} {Colors.GREEN}{format_uptime(status.get('UpTime', 0)):<{width-15}}{Colors.CYAN} ║")
    print(f"╠{'═' * width}╣")

    # Performance metrics with bars
    bar_width = 25
    
    cpu_usage = status.get('cpu_usageinfo', {}).get('CPUUsage', 0)
    cpu_filled = int(cpu_usage / 100 * bar_width)
    cpu_color = Colors.GREEN if cpu_usage < 50 else Colors.YELLOW if cpu_usage < 80 else Colors.RED
    cpu_bar = cpu_color + "█" * cpu_filled + Colors.RESET + "░" * (bar_width - cpu_filled)
    cpu_label = f"{'CPU:':<8}{cpu_usage:>3.0f}%"
    cpu_spaces = width - 2 - len(cpu_label) - 1 - bar_width
    print(f"{Colors.CYAN}║ {Colors.RESET}{cpu_label}{' ' * cpu_spaces} {cpu_bar} {Colors.CYAN}║")

    mem_info = status.get('mem_info', {})
    if mem_info:
        total_mb, used_mb, free_mb, used_percent = format_memory(mem_info.get('Total', 0), mem_info.get('Free', 0))
        mem_filled = int(used_percent / 100 * bar_width)
        mem_color = Colors.GREEN if used_percent < 50 else Colors.YELLOW if used_percent < 80 else Colors.RED
        mem_bar = mem_color + "█" * mem_filled + Colors.RESET + "░" * (bar_width - mem_filled)
        mem_label = f"{'Memory:':<8}{used_percent:>3.0f}% ({used_mb:.0f}/{total_mb:.0f}MB)"
        mem_spaces = width - 2 - len(mem_label) - 1 - bar_width
        print(f"{Colors.CYAN}║ {Colors.RESET}{mem_label}{' ' * mem_spaces} {mem_bar} {Colors.CYAN}║")

    print(f"╚{'═' * width}╝{Colors.RESET}\n")


def login_both_gateways():
    """Authenticate to both gateways and display status"""
    print(f"\n{Colors.BOLD}🌐 Nokia FastMile 5G Gateway Client{Colors.RESET}")
    print("━" * 40)
    
    results = []
    gateways = [('ODU', '192.168.0.1', login_odu), ('IDU', '192.168.1.1', login_idu)]
    
    for gateway_type, ip, login_func in gateways:
        print(f"\n{Colors.BLUE}🔍 Connecting to {gateway_type} Gateway at {ip}...{Colors.RESET}")
        result, sess = login_func()
        
        if result.get('result') == 0:
            print(f"\n{Colors.GREEN}✅ {gateway_type} Authentication successful!{Colors.RESET}")
            print(f"{Colors.BLUE}🔑 Session: {Colors.CYAN}{result.get('sid', 'N/A')}{Colors.RESET}")
            print(f"{Colors.BLUE}🎫 Token: {Colors.CYAN}{result.get('token', 'N/A')}{Colors.RESET}")
            results.append((gateway_type, ip, result, sess))
            
            status = get_device_status(sess, ip)
            if status:
                print_status_box(status, gateway_type, ip)
            else:
                print(f"{Colors.RED}❌ Failed to retrieve device status{Colors.RESET}")
        else:
            print(f"\n{Colors.RED}❌ {gateway_type} Authentication failed{Colors.RESET}")
            print(f"{Colors.RED}   Error: {Colors.YELLOW}{result.get('error', 'Unknown')}{Colors.RESET}")
    
    # Cleanup sessions
    for _, ip, _, sess in results:
        if sess:
            try:
                sess.get(f"https://{ip}:443/login_web_app.cgi?out", timeout=10, verify=False)
            except:
                pass
    
    return results


if __name__ == "__main__":
    try:
        results = login_both_gateways()
        
        print(f"\n{Colors.BOLD}📊 Summary{Colors.RESET}")
        print("━" * 20)
        
        successful = [r for r in results if r[2].get('result') == 0]
        
        if successful:
            print(f"{Colors.GREEN}✅ Successful: {len(successful)}/2{Colors.RESET}")
            for gateway_type, ip, result, _ in successful:
                token = result.get('token', 'N/A')
                sid = result.get('sid', 'N/A')
                print(f"  {Colors.CYAN}{gateway_type} ({ip}){Colors.RESET}: Token={Colors.YELLOW}{token}{Colors.RESET}, SID={Colors.YELLOW}{sid}{Colors.RESET}")
        else:
            print(f"{Colors.RED}❌ No successful connections{Colors.RESET}")
        print()
            
    except KeyboardInterrupt:
        print(f"\n{Colors.YELLOW}Operation cancelled{Colors.RESET}")
    except Exception as e:
        print(f"\n{Colors.RED}Error: {str(e)}{Colors.RESET}")
        sys.exit(1)

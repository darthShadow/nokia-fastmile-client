package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type Client struct {
	BaseURL     string
	GatewayIP   string
	GatewayType string
	HTTPClient  *http.Client
	Token       string
	SID         string
	LoggedIn    bool
}

type LoginResponse struct {
	Result int    `json:"result"`
	Token  string `json:"token"`
	SID    string `json:"sid"`
}

type NonceResponse struct {
	Nonce      string `json:"nonce"`
	RandomKey  string `json:"randomKey"`
	Iterations int    `json:"iterations"`
	PubKey     string `json:"pubkey"`
}

type SaltResponse struct {
	Alati string `json:"alati"`
}

type DeviceStatus struct {
	ModelName       string `json:"ModelName"`
	SerialNumber    string `json:"SerialNumber"`
	SoftwareVersion string `json:"SoftwareVersion"`
	UpTime          int    `json:"UpTime"`
	CPUUsageInfo    struct {
		CPUUsage int `json:"CPUUsage"`
	} `json:"cpu_usageinfo"`
	MemInfo struct {
		Total int `json:"Total"`
		Free  int `json:"Free"`
	} `json:"mem_info"`
}

func NewClient(gatewayIP string, useHTTPS bool) *Client {
	protocol := "http"
	if useHTTPS {
		protocol = "https"
	}

	tr := &http.Transport{}
	if useHTTPS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		Jar:       jar,
	}

	baseURL := fmt.Sprintf("%s://%s:443", protocol, gatewayIP)

	gatewayType := "IDU"
	if gatewayIP == "192.168.0.1" {
		gatewayType = "ODU"
	}

	c := &Client{
		BaseURL:     baseURL,
		GatewayIP:   gatewayIP,
		GatewayType: gatewayType,
		HTTPClient:  client,
	}

	c.setDefaultHeaders()

	return c
}

func (c *Client) setDefaultHeaders() {
	existingTransport := c.HTTPClient.Transport
	if existingTransport == nil {
		existingTransport = http.DefaultTransport
	}

	c.HTTPClient.Transport = &headerTransport{
		RoundTripper: existingTransport,
		headers: map[string]string{
			"Accept":        "application/json, text/plain, */*",
			"User-Agent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
			"Cache-Control": "no-cache",
			"Pragma":        "no-cache",
			"Origin":        c.BaseURL,
			"Referer":       c.BaseURL + "/",
		},
	}
}

type headerTransport struct {
	http.RoundTripper
	headers map[string]string
}

func (ht *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range ht.headers {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}
	return ht.RoundTripper.RoundTrip(req)
}

// Cryptographic helper functions for ODU authentication
func base64URLEscape(b64 string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(b64, "=", "."), "/", "_"), "+", "-")
}

func sha256Hash(val1, val2 string) string {
	h := sha256.Sum256([]byte(val1 + ":" + val2))
	return base64.StdEncoding.EncodeToString(h[:])
}

func sha256Single(val string) string {
	h := sha256.Sum256([]byte(val))
	return fmt.Sprintf("%x", h)
}

func sha256URL(val1, val2 string) string {
	return base64URLEscape(sha256Hash(val1, val2))
}

func randomWords(numWords int) string {
	bytes := make([]byte, numWords*4)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func (c *Client) InitializeSession() error {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/")
	if err != nil {
		return fmt.Errorf("failed to initialize session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session initialization failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Login() error {
	return c.LoginWithProgress(false, nil)
}

func (c *Client) LoginWithProgress(showProgress bool, logger *log.Logger) error {
	if c.GatewayType == "ODU" {
		return c.LoginODUWithProgress(showProgress, logger)
	}
	return c.LoginIDUWithProgress(showProgress, logger)
}

func (c *Client) LoginODU() error {
	return c.LoginODUWithProgress(false, nil)
}

func (c *Client) LoginIDU() error {
	return c.LoginIDUWithProgress(false, nil)
}

func (c *Client) LoginODUWithProgress(showProgress bool, logger *log.Logger) error {
	username := "admin"
	password := "ANKODACF00005930"

	if showProgress {
		fmt.Printf("  \033[94mStep 1:\033[0m Initializing Session...\n")
	} else if logger != nil {
		logger.Info("Initializing Session", "step", "1")
	}
	if err := c.InitializeSession(); err != nil {
		return err
	}
	if showProgress {
		fmt.Printf("  \033[92m✓\033[0m Session Initialized\n")
	} else if logger != nil {
		logger.Info("Session Initialized")
	}

	if showProgress {
		fmt.Printf("  \033[94mStep 2:\033[0m Clearing Existing Sessions...\n")
	} else if logger != nil {
		logger.Info("Clearing Existing Sessions", "step", "2")
	}
	c.HTTPClient.Get(c.BaseURL + "/login_web_app.cgi?out")

	// Get nonce
	if showProgress {
		fmt.Printf("  \033[94mStep 3:\033[0m Getting Nonce...\n")
	} else if logger != nil {
		logger.Info("Getting Nonce", "step", "3")
	}
	resp, err := c.HTTPClient.Get(c.BaseURL + "/login_web_app.cgi?nonce")
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}
	defer resp.Body.Close()

	var nonceResp NonceResponse
	if err := json.NewDecoder(resp.Body).Decode(&nonceResp); err != nil {
		return fmt.Errorf("invalid nonce response: %w", err)
	}
	if showProgress {
		fmt.Printf("  \033[92m✓\033[0m Nonce: \033[96m%s...\033[0m\n", nonceResp.Nonce[:20])
	} else if logger != nil {
		logger.Info("Nonce Received", "preview", nonceResp.Nonce[:20]+"...")
	}

	// Get salt
	if showProgress {
		fmt.Printf("  \033[94mStep 4:\033[0m Getting Salt...\n")
	} else if logger != nil {
		logger.Info("Getting Salt", "step", "4")
	}
	userhash := sha256URL(username, nonceResp.Nonce)
	saltData := fmt.Sprintf("userhash=%s&nonce=%s", userhash, base64URLEscape(nonceResp.Nonce))

	req, err := http.NewRequest("POST", c.BaseURL+"/login_web_app.cgi?salt", strings.NewReader(saltData))
	if err != nil {
		return fmt.Errorf("failed to create salt request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get salt: %w", err)
	}
	defer resp.Body.Close()

	var saltResp SaltResponse
	if err := json.NewDecoder(resp.Body).Decode(&saltResp); err != nil {
		return fmt.Errorf("invalid salt response: %w", err)
	}
	if showProgress {
		saltPreview := saltResp.Alati
		if len(saltPreview) > 20 {
			saltPreview = saltPreview[:20] + "..."
		}
		fmt.Printf("  \033[92m✓\033[0m Salt: \033[96m%s\033[0m\n", saltPreview)
	} else if logger != nil {
		saltPreview := saltResp.Alati
		if len(saltPreview) > 20 {
			saltPreview = saltPreview[:20] + "..."
		}
		logger.Info("Salt Received", "preview", saltPreview)
	}

	// Process password
	if showProgress {
		fmt.Printf("  \033[94mStep 5:\033[0m Processing Authentication...\n")
	} else if logger != nil {
		logger.Info("Processing Authentication", "step", "5")
	}
	passHash := saltResp.Alati + password
	if nonceResp.Iterations >= 1 {
		passHash = sha256Single(passHash)
	}

	// Generate authentication response
	loginHash := sha256Hash(username, strings.ToLower(passHash))
	response := sha256URL(loginHash, nonceResp.Nonce)
	randomKeyHash := sha256URL(nonceResp.RandomKey, nonceResp.Nonce)
	enckey := randomWords(4)
	enciv := randomWords(4)

	// Submit authentication
	if showProgress {
		fmt.Printf("  \033[94mStep 6:\033[0m Submitting Authentication...\n")
	} else if logger != nil {
		logger.Info("Submitting Authentication", "step", "6")
	}
	authData := fmt.Sprintf("userhash=%s&RandomKeyhash=%s&response=%s&nonce=%s&enckey=%s&enciv=%s",
		userhash, randomKeyHash, response, base64URLEscape(nonceResp.Nonce),
		base64URLEscape(enckey), base64URLEscape(enciv))

	req, err = http.NewRequest("POST", c.BaseURL+"/login_web_app.cgi", strings.NewReader(authData))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("invalid login response: %w", err)
	}

	if loginResp.Result != 0 {
		return fmt.Errorf("gateway error code %d", loginResp.Result)
	}

	c.Token = loginResp.Token
	c.SID = loginResp.SID
	c.LoggedIn = true

	return nil
}

func (c *Client) LoginIDUWithProgress(showProgress bool, logger *log.Logger) error {
	if showProgress {
		fmt.Printf("  \033[94mStep 1:\033[0m Initializing Session...\n")
	} else if logger != nil {
		logger.Info("Initializing Session", "step", "1")
	}
	if err := c.InitializeSession(); err != nil {
		return err
	}
	if showProgress {
		fmt.Printf("  \033[92m✓\033[0m Session Initialized\n")
	} else if logger != nil {
		logger.Info("Session Initialized")
	}

	if showProgress {
		fmt.Printf("  \033[94mStep 2:\033[0m Clearing Existing Sessions...\n")
	} else if logger != nil {
		logger.Info("Clearing Existing Sessions", "step", "2")
	}
	c.HTTPClient.Get(c.BaseURL + "/login_web_app.cgi?out")

	if showProgress {
		fmt.Printf("  \033[94mStep 3:\033[0m Processing Authentication...\n")
	} else if logger != nil {
		logger.Info("Processing Authentication", "step", "3")
	}
	// Browser-captured encrypted payload for IDU
	browserPayload := "encrypted=1&ct=DiVgETIqDqEOAr6WsF4-kX2yYqyEp1KnZxC5j5__HGCAztvljLzKvNQwuPI25mqrteWc7D63ivOBANHyD6SveoIQc9-9wjfaEhTZzVd-rJlbhE-O5V9kpXdRavvHhBbReCZLmk2wlOPFshOO85dBhPmmi0B0N3maAa6bF9GS-rNRByE4-QP4CODsKa9lEaQ7qmy3aLq43mAtP3hELrulRxnkKbGC0Yk-9VSIftRe0Uw3zyFhyYjNIJnCT3CjsJTH-gSVlxvHwJukztsE0XwfBQ&ck=fewEnnPAQ2ApoDmGZKGuy9mVhU7jozMgIdf3FAfsjjClcqlsOwDJgPp1iR4It-R4tmZOu_OmgKl4Vg1OpK6jgOFMZ-Mh0HDMnb4fL8uOO-rQolJG2tNeYKZvluYj9KM7-rzpz1mKHKaQ9GPS37avrkBNxiYDZityySUR66CBT9Q."

	if showProgress {
		fmt.Printf("  \033[94mStep 4:\033[0m Submitting Authentication...\n")
	} else if logger != nil {
		logger.Info("Submitting Authentication", "step", "4")
	}
	req, err := http.NewRequest("POST", c.BaseURL+"/login_web_app.cgi", strings.NewReader(browserPayload))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Origin", c.BaseURL)
	req.Header.Set("Referer", c.BaseURL+"/")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	if loginResp.Result != 0 {
		return fmt.Errorf("gateway error code %d", loginResp.Result)
	}

	c.Token = loginResp.Token
	c.SID = loginResp.SID
	c.LoggedIn = true

	return nil
}

func (c *Client) GetDeviceStatus() (*DeviceStatus, error) {
	if !c.LoggedIn {
		return nil, fmt.Errorf("not logged in")
	}

	resp, err := c.HTTPClient.Get(c.BaseURL + "/device_status_web_app.cgi?getroot")
	if err != nil {
		return nil, fmt.Errorf("failed to get device status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device status request failed with status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	contentStr := strings.TrimSpace(string(bodyBytes))

	var status DeviceStatus
	if err := json.Unmarshal([]byte(contentStr), &status); err == nil {
		return &status, nil
	}
	contentStr = strings.ReplaceAll(contentStr, ",]", "]")
	contentStr = strings.ReplaceAll(contentStr, ",}", "}")

	// Handle malformed JSON with multiple commas
	re1 := regexp.MustCompile(`,\s*,`)
	contentStr = re1.ReplaceAllString(contentStr, ",")

	re2 := regexp.MustCompile(`\[\s*,`)
	contentStr = re2.ReplaceAllString(contentStr, "[")

	if err := json.Unmarshal([]byte(contentStr), &status); err != nil {
		if len(contentStr) > 200 {
			return nil, fmt.Errorf("failed to decode device status: %w (content starts with: %.200s...)", err, contentStr)
		}
		return nil, fmt.Errorf("failed to decode device status: %w (content: %s)", err, contentStr)
	}

	return &status, nil
}

func (c *Client) Logout() error {
	if !c.LoggedIn {
		return nil
	}

	_, err := c.HTTPClient.Get(c.BaseURL + "/login_web_app.cgi?out")
	if err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	c.Token = ""
	c.SID = ""
	c.LoggedIn = false

	return nil
}

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

const (
	ORBI_GATEWAY_IP = "192.168.10.254"
	ORBI_USERNAME   = "admin"
	ORBI_PASSWORD   = "Home-404_BOM"

	DEV_DEVICE_INFO_PATH = "/DEV_device_info.htm"
	REBOOT_PATH          = "/reboot.htm"
	APPLY_CGI_PATH       = "/apply.cgi"
)

type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
	Logger     *log.Logger
}

type Device struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	MAC         string `json:"mac"`
	ConnType    string `json:"conn_type"`
	BackhaulSta string `json:"backhaul_sta,omitempty"`
}

type DeviceInfo struct {
	ConnectedDevices []Device
	ActiveDevices    []Device
	InactiveDevices  []Device
	TotalCount       int
}

func NewClient(logger *log.Logger) *Client {
	baseURL := fmt.Sprintf("http://%s", ORBI_GATEWAY_IP)
	jar, _ := cookiejar.New(nil)

	return &Client{
		BaseURL:  baseURL,
		Username: ORBI_USERNAME,
		Password: ORBI_PASSWORD,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		Logger: logger,
	}
}

func (c *Client) GetDevices() (*DeviceInfo, error) {
	timestamp := time.Now().Unix()
	requestURL := fmt.Sprintf("%s%s?ts=%d", c.BaseURL, DEV_DEVICE_INFO_PATH, timestamp)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("User-Agent", "NetgearOrbiClient/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	bodyBytes := make([]byte, 0, resp.ContentLength)
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	body := string(bodyBytes)

	lines := strings.SplitSeq(body, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "device="); ok {
			deviceJSON := after

			var devices []Device
			if err := json.Unmarshal([]byte(deviceJSON), &devices); err != nil {
				return nil, fmt.Errorf("failed to parse device JSON: %w", err)
			}

			return c.processDevices(devices), nil
		}
	}

	return nil, fmt.Errorf("device data not found in response")
}

func (c *Client) processDevices(devices []Device) *DeviceInfo {
	info := &DeviceInfo{
		ConnectedDevices: devices,
		TotalCount:       len(devices),
	}

	for _, device := range devices {
		if device.BackhaulSta == "Good" {
			info.ActiveDevices = append(info.ActiveDevices, device)
		} else {
			info.InactiveDevices = append(info.InactiveDevices, device)
		}
	}

	return info
}

func (c *Client) getTimestampFromRebootPage() (string, error) {
	requestURL := fmt.Sprintf("%s%s", c.BaseURL, REBOOT_PATH)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	bodyBytes := make([]byte, 0, resp.ContentLength)
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			bodyBytes = append(bodyBytes, buffer[:n]...)
		}
		if err != nil {
			break
		}
	}

	body := string(bodyBytes)

	timestampRegex := regexp.MustCompile(`timestamp=(\d+)"`)
	matches := timestampRegex.FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract timestamp from reboot page")
	}

	return matches[1], nil
}

func (c *Client) RebootRouter() error {
	timestamp, err := c.getTimestampFromRebootPage()
	if err != nil {
		return fmt.Errorf("failed to get timestamp: %w", err)
	}

	requestURL := fmt.Sprintf("%s%s?/reboot_waiting.htm timestamp=%s",
		c.BaseURL, APPLY_CGI_PATH, timestamp)

	formData := url.Values{
		"submit_flag": {"reboot"},
		"yes":         {"Yes"},
	}

	req, err := http.NewRequest("POST", requestURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", fmt.Sprintf("%s%s", c.BaseURL, REBOOT_PATH))
	req.Header.Set("User-Agent", "NetgearOrbiClient/1.0")

	c.Logger.Infof("Initiating router reboot...")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("reboot request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reboot failed with status %d", resp.StatusCode)
	}

	return nil
}

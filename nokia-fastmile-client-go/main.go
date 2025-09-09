package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

func main() {
	var (
		useHTTPS = flag.Bool("https", true, "Use HTTPS (default: true)")
		pretty   = flag.Bool("pretty", false, "Enable pretty output with styling")
		verbose  = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	isInTerminal := !ShouldUsePlainOutput()
	usePrettyOutput := *pretty && isInTerminal

	logger := log.New(os.Stderr)
	if *verbose {
		logger.SetLevel(log.DebugLevel)
	} else if usePrettyOutput {
		logger.SetLevel(log.ErrorLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}

	// Enhance log colors for better visibility
	styles := log.DefaultStyles()
	styles.Levels[log.DebugLevel] = lipgloss.NewStyle().
		SetString("DEBU").
		Foreground(lipgloss.Color("27")).Bold(true)
	styles.Levels[log.InfoLevel] = lipgloss.NewStyle().
		SetString("INFO").
		Foreground(lipgloss.Color("28")).Bold(true)
	styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
		SetString("WARN").
		Foreground(lipgloss.Color("208")).Bold(true)
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString("ERRO").
		Foreground(lipgloss.Color("124")).Bold(true)

	styles.Key = lipgloss.NewStyle().Foreground(lipgloss.Color("135")).Bold(true)
	styles.Value = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	logger.SetStyles(styles)

	if usePrettyOutput {
		fmt.Print(RenderHeader())
	}

	// Try both gateways
	gateways := []string{"192.168.0.1", "192.168.1.1"}
	var successfulResults []struct {
		client *Client
		status *DeviceStatus
	}

	for _, gatewayIP := range gateways {
		client := NewClient(gatewayIP, *useHTTPS)

		if usePrettyOutput {
			fmt.Printf("\n🔍 Connecting to %s Gateway at %s...\n", client.GatewayType, gatewayIP)
		} else {
			logger.Info("Attempting Connection", "gateway-type", client.GatewayType, "ip", gatewayIP)
		}

		logger.Debug("Attempting To Login...")
		if err := client.LoginWithProgress(usePrettyOutput, logger); err != nil {
			errorMsg := err.Error()

			if usePrettyOutput {
				fmt.Printf("\n%s\n", RenderErrorLipgloss(fmt.Sprintf("%s Authentication Failed", client.GatewayType)))

				errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).MarginLeft(3)
				warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).MarginLeft(3)

				if strings.Contains(errorMsg, "HTTP error") {
					fmt.Println(errStyle.Render("HTTP status: " + strings.TrimPrefix(errorMsg, "HTTP error ")))
				} else if strings.Contains(errorMsg, "gateway error code") {
					code := strings.TrimPrefix(errorMsg, "gateway error code ")
					fmt.Println(errStyle.Render("Gateway error code: " + code))
				} else if strings.Contains(errorMsg, "invalid JSON") {
					fmt.Println(errStyle.Render("Invalid response from gateway"))
					fmt.Println(warnStyle.Render("💡 Hint: Payload may be expired. Generate a new one from browser."))
				} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "connection") {
					fmt.Println(errStyle.Render("Unable to connect to gateway"))
				} else {
					fmt.Println(errStyle.Render("Error: " + errorMsg))
				}
			} else {
				logger.Error("Authentication Failed", "gateway-type", client.GatewayType, "error", errorMsg)
			}
			continue
		}

		if usePrettyOutput {
			fmt.Printf("\n%s\n", RenderSuccessLipgloss(fmt.Sprintf("%s Authentication Successful!", client.GatewayType)))
			if client.SID != "" {
				fmt.Printf("%s\n", RenderInfoLipgloss(client.SID))
			}
			if client.Token != "" {
				fmt.Printf("%s\n", RenderTokenLipgloss(client.Token))
			}
			fmt.Println() // Add spacing before the table
		} else {
			logger.Info("Authentication Successful", "gateway-type", client.GatewayType)
			if client.SID != "" {
				logger.Info("Session ID Received", "session-id", client.SID)
			}
			if client.Token != "" {
				logger.Info("Token Received", "token", client.Token)
			}
		}

		logger.Debug("Fetching Device Status...")
		status, err := client.GetDeviceStatus()
		if err != nil {
			if usePrettyOutput {
				fmt.Printf("\n%s\n", RenderErrorLipgloss("Failed To Retrieve Device Status"))
				errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).MarginLeft(3)
				fmt.Println(errStyle.Render("Device may be unreachable or session expired"))
			} else {
				logger.Error("Failed To Retrieve Device Status", "error", err.Error())
			}

			client.Logout()
			continue
		}

		successfulResults = append(successfulResults, struct {
			client *Client
			status *DeviceStatus
		}{client, status})

		if usePrettyOutput {
			fmt.Print(RenderStatusBoxLipglossWithType(status, client.GatewayType, gatewayIP))
		} else {
			logger.Info("Device Status Retrieved", "gateway-type", client.GatewayType)
			logger.Info("Device Model", "model", status.ModelName)
			logger.Info("Device Serial", "serial", status.SerialNumber)
			logger.Info("Software Version", "version", status.SoftwareVersion)
			logger.Info("Device Uptime", "uptime", FormatUptime(status.UpTime))
			logger.Info("CPU Usage", "cpu-percent", status.CPUUsageInfo.CPUUsage)

			if status.MemInfo.Total > 0 {
				memInfo := FormatMemory(status.MemInfo.Total, status.MemInfo.Free)
				logger.Info("Memory Usage",
					"memory-percent", fmt.Sprintf("%.0f", memInfo.UsedPercent),
					"memory-used-mb", fmt.Sprintf("%.0f", memInfo.UsedMB),
					"memory-total-mb", fmt.Sprintf("%.0f", memInfo.TotalMB))
			}
		}
	}

	if usePrettyOutput {
		// Summary header - bold like Python
		summaryStyle := lipgloss.NewStyle().Bold(true)
		fmt.Printf("\n%s\n", summaryStyle.Render("📊 Summary"))

		if len(successfulResults) > 0 {
			// Success count - green like Python
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("40")) // Green like our success messages
			fmt.Printf("%s\n", successStyle.Render(fmt.Sprintf("✅ Successful: %d/2", len(successfulResults))))

			// Connection details
			for _, result := range successfulResults {
				token := result.client.Token
				if token == "" {
					token = "N/A"
				}
				sid := result.client.SID
				if sid == "" {
					sid = "N/A"
				}

				gatewayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)   // Cyan, bold
				ipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("135"))                  // Purple for IP
				labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)     // Bright cyan for labels - stark contrast with orange
				tokenSidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true) // Orange for values - stark contrast

				fmt.Printf("  %s %s: %s%s, %s%s\n",
					gatewayStyle.Render(result.client.GatewayType),
					ipStyle.Render("("+result.client.GatewayIP+")"),
					labelStyle.Render("Token="),
					tokenSidStyle.Render(token),
					labelStyle.Render("SID="),
					tokenSidStyle.Render(sid))
			}
		} else {
			// No connections - red like Python
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
			fmt.Printf("%s\n", errorStyle.Render("❌ No successful connections"))
		}
		fmt.Println()
	}

	// Logout from all successful sessions
	logger.Debug("Logging Out From Successful Sessions...")
	for _, result := range successfulResults {
		if err := result.client.Logout(); err != nil {
			logger.Error("Logout Failed", "gateway-type", result.client.GatewayType, "error", err)
		}
	}
}

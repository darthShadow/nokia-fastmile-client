package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var version = "1.0.0"

func main() {
	var (
		command     = flag.String("cmd", "list", "Command to execute: list, reboot")
		pretty      = flag.Bool("pretty", false, "Enable pretty output with styling")
		verbose     = flag.Bool("verbose", false, "Enable verbose logging")
		force       = flag.Bool("force", false, "Skip confirmation prompts")
		showVersion = flag.Bool("version", false, "Show version information")
		help        = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Netgear Orbi Client v%s\n", version)
		return
	}

	if *help {
		showHelp()
		return
	}

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

	// Configure log colors
	if usePrettyOutput {
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
	}

	client := NewClient(logger)

	switch strings.ToLower(*command) {
	case "list", "devices":
		handleListCommand(client, usePrettyOutput)
	case "reboot", "restart":
		handleRebootCommand(client, *force, usePrettyOutput)
	default:
		DisplayError(fmt.Sprintf("Unknown command: %s", *command), usePrettyOutput)
		fmt.Fprintf(os.Stderr, "\nAvailable commands: list, reboot\n")
		fmt.Fprintf(os.Stderr, "Use -help for more information.\n")
		os.Exit(1)
	}
}

func handleListCommand(client *Client, usePrettyOutput bool) {
	if !usePrettyOutput {
		fmt.Fprintf(os.Stderr, "Fetching device information from router...\n")
	}

	devices, err := client.GetDevices()
	if err != nil {
		DisplayError(fmt.Sprintf("Failed to get devices: %s", err), usePrettyOutput)
		os.Exit(1)
	}

	if devices == nil || devices.TotalCount == 0 {
		DisplayInfo("No devices found or unable to connect to router", usePrettyOutput)
		return
	}

	DisplayDeviceInfo(devices, usePrettyOutput)
}

func handleRebootCommand(client *Client, force bool, usePrettyOutput bool) {
	if !force {
		if !confirmReboot(usePrettyOutput) {
			DisplayInfo("Reboot cancelled.", usePrettyOutput)
			return
		}
	} else {
		DisplayInfo("Force mode: Skipping confirmation prompt", usePrettyOutput)
	}

	if err := client.RebootRouter(); err != nil {
		DisplayError(fmt.Sprintf("Failed to reboot router: %s", err), usePrettyOutput)
		os.Exit(1)
	}

	DisplayRebootSuccess(usePrettyOutput)
}

func confirmReboot(usePrettyOutput bool) bool {
	var prompt string
	if usePrettyOutput {
		warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
		prompt = warningStyle.Render("Are you sure you want to reboot the router? (yes/no): ")
	} else {
		prompt = "Are you sure you want to reboot the router? (yes/no): "
	}

	fmt.Print(prompt)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "yes" || response == "y"
	}

	return false
}

func showHelp() {
	fmt.Printf("Netgear Orbi Client v%s\n\n", version)
	fmt.Println("USAGE:")
	fmt.Println("  netgear-orbi-go [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -cmd string       Command to execute: list, reboot (default \"list\")")
	fmt.Println("  -pretty           Enable pretty output with styling")
	fmt.Println("  -verbose          Enable verbose logging")
	fmt.Println("  -force            Skip confirmation prompts")
	fmt.Println("  -version          Show version information")
	fmt.Println("  -help             Show this help message")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  list, devices     List all connected devices (default)")
	fmt.Println("  reboot, restart   Reboot the router")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # List devices")
	fmt.Println("  netgear-orbi-go")
	fmt.Println()
	fmt.Println("  # List devices with pretty output")
	fmt.Println("  netgear-orbi-go -pretty")
	fmt.Println()
	fmt.Println("  # Reboot router without confirmation")
	fmt.Println("  netgear-orbi-go -cmd reboot -force")
}

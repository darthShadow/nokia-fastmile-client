package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Color scheme
	primaryColor   = lipgloss.Color("75")  // Light green
	secondaryColor = lipgloss.Color("69")  // Light blue
	accentColor    = lipgloss.Color("208") // Orange
	errorColor     = lipgloss.Color("124") // Red
	warningColor   = lipgloss.Color("226") // Yellow
	mutedColor     = lipgloss.Color("243") // Gray

	// Styles
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true).
			Underline(true)

	deviceNameStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Width(30).
			Align(lipgloss.Left)

	deviceIPStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // Cyan
			Width(15).
			Align(lipgloss.Left)

	deviceMACStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(17).
			Align(lipgloss.Left)

	deviceTypeStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Width(10).
			Align(lipgloss.Left)

	activeStatusStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	inactiveStatusStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	separatorStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)

func ShouldUsePlainOutput() bool {
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return true
	}

	if !isatty() {
		return true
	}

	return false
}

func isatty() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func DisplayDeviceInfo(info *DeviceInfo, usePrettyOutput bool) {
	if usePrettyOutput {
		displayDeviceInfoStyled(info)
	} else {
		displayDeviceInfoPlain(info)
	}
}

func displayDeviceInfoStyled(info *DeviceInfo) {
	// Title
	title := titleStyle.Render("NETGEAR Orbi Router - Connected Devices")
	fmt.Println(title)
	fmt.Println()

	// Separator
	separator := separatorStyle.Render(strings.Repeat("=", 80))
	fmt.Println(separator)
	fmt.Println()

	// Device count
	countInfo := infoStyle.Render(fmt.Sprintf("Total devices: %d", info.TotalCount))
	fmt.Println(countInfo)
	fmt.Println()

	// Active devices section
	if len(info.ActiveDevices) > 0 {
		activeHeader := headerStyle.Render("Active Devices")
		fmt.Println(activeHeader)

		subSeparator := separatorStyle.Render(strings.Repeat("-", 80))
		fmt.Println(subSeparator)

		// Sort devices by name
		sortedActive := make([]Device, len(info.ActiveDevices))
		copy(sortedActive, info.ActiveDevices)
		sort.Slice(sortedActive, func(i, j int) bool {
			return strings.ToLower(sortedActive[i].Name) < strings.ToLower(sortedActive[j].Name)
		})

		for _, device := range sortedActive {
			displayDeviceStyled(device)
		}
		fmt.Println()
	}

	// Inactive/Other devices section
	if len(info.InactiveDevices) > 0 {
		var sectionTitle string
		if len(info.ActiveDevices) > 0 {
			sectionTitle = "Other Devices"
		} else {
			sectionTitle = "Devices"
		}

		otherHeader := headerStyle.Render(sectionTitle)
		fmt.Println(otherHeader)

		subSeparator := separatorStyle.Render(strings.Repeat("-", 80))
		fmt.Println(subSeparator)

		// Sort devices by name
		sortedInactive := make([]Device, len(info.InactiveDevices))
		copy(sortedInactive, info.InactiveDevices)
		sort.Slice(sortedInactive, func(i, j int) bool {
			return strings.ToLower(sortedInactive[i].Name) < strings.ToLower(sortedInactive[j].Name)
		})

		for _, device := range sortedInactive {
			displayDeviceStyled(device)
		}
		fmt.Println()
	}

	// Bottom separator
	fmt.Println(separator)
}

func displayDeviceStyled(device Device) {
	name := deviceNameStyle.Render(truncateString(device.Name, 28))
	ip := deviceIPStyle.Render(fmt.Sprintf("IP: %s", device.IP))
	mac := deviceMACStyle.Render(fmt.Sprintf("MAC: %s", device.MAC))
	connType := deviceTypeStyle.Render(fmt.Sprintf("Type: %s", device.ConnType))

	var status string
	if device.BackhaulSta == "Good" {
		status = activeStatusStyle.Render("Status: Active (Good)")
	} else if device.BackhaulSta == "Poor" {
		status = inactiveStatusStyle.Render("Status: Active (Poor)")
	} else if device.BackhaulSta != "" {
		status = inactiveStatusStyle.Render(fmt.Sprintf("Status: Active (%s)", device.BackhaulSta))
	} else {
		status = inactiveStatusStyle.Render("Status: Connected")
	}

	line := lipgloss.JoinHorizontal(
		lipgloss.Top,
		name,
		ip,
		mac,
		connType,
		status,
	)

	fmt.Println(line)
}

func displayDeviceInfoPlain(info *DeviceInfo) {
	fmt.Println("NETGEAR Orbi Router - Connected Devices")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nTotal devices: %d\n\n", info.TotalCount)

	// Active devices
	if len(info.ActiveDevices) > 0 {
		fmt.Println("Active Devices:")
		fmt.Println(strings.Repeat("-", 80))

		sortedActive := make([]Device, len(info.ActiveDevices))
		copy(sortedActive, info.ActiveDevices)
		sort.Slice(sortedActive, func(i, j int) bool {
			return strings.ToLower(sortedActive[i].Name) < strings.ToLower(sortedActive[j].Name)
		})

		for _, device := range sortedActive {
			displayDevicePlain(device)
		}
		fmt.Println()
	}

	// Inactive devices
	if len(info.InactiveDevices) > 0 {
		var sectionTitle string
		if len(info.ActiveDevices) > 0 {
			sectionTitle = "Other Devices:"
		} else {
			sectionTitle = "Devices:"
		}

		fmt.Println(sectionTitle)
		fmt.Println(strings.Repeat("-", 80))

		sortedInactive := make([]Device, len(info.InactiveDevices))
		copy(sortedInactive, info.InactiveDevices)
		sort.Slice(sortedInactive, func(i, j int) bool {
			return strings.ToLower(sortedInactive[i].Name) < strings.ToLower(sortedInactive[j].Name)
		})

		for _, device := range sortedInactive {
			displayDevicePlain(device)
		}
		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
}

func displayDevicePlain(device Device) {
	name := truncateString(device.Name, 28)
	ip := device.IP
	mac := device.MAC
	connType := device.ConnType

	var status string
	if device.BackhaulSta == "Good" {
		status = "Active (Good)"
	} else if device.BackhaulSta == "Poor" {
		status = "Active (Poor)"
	} else if device.BackhaulSta != "" {
		status = fmt.Sprintf("Active (%s)", device.BackhaulSta)
	} else {
		status = "Connected"
	}

	fmt.Printf("%-30s IP: %-15s MAC: %-17s Type: %-10s Status: %s\n",
		name, ip, mac, connType, status)
}

func DisplayRebootSuccess(usePrettyOutput bool) {
	if usePrettyOutput {
		displayRebootSuccessStyled()
	} else {
		displayRebootSuccessPlain()
	}
}

func displayRebootSuccessStyled() {
	checkmark := successStyle.Render("✓")
	message := successStyle.Render("Reboot command sent successfully!")

	fmt.Printf("%s %s\n\n", checkmark, message)

	infoLines := []string{
		"The router is now rebooting. This typically takes 2-3 minutes.",
		"Your internet connection will be temporarily unavailable.",
		"",
		"You can check if the router is back online by:",
		"1. Pinging the router: ping 192.168.1.1",
		"2. Running this tool again to list devices",
	}

	for _, line := range infoLines {
		if line == "" {
			fmt.Println()
		} else {
			styled := infoStyle.Render(line)
			fmt.Println(styled)
		}
	}
}

func displayRebootSuccessPlain() {
	fmt.Println("✓ Reboot command sent successfully!")
	fmt.Println()
	fmt.Println("The router is now rebooting. This typically takes 2-3 minutes.")
	fmt.Println("Your internet connection will be temporarily unavailable.")
	fmt.Println()
	fmt.Println("You can check if the router is back online by:")
	fmt.Println("1. Pinging the router: ping 192.168.1.1")
	fmt.Println("2. Running this tool again to list devices")
}

func DisplayError(message string, usePrettyOutput bool) {
	if usePrettyOutput {
		errorIcon := errorStyle.Render("✗")
		styledMessage := errorStyle.Render(message)
		fmt.Printf("%s %s\n", errorIcon, styledMessage)
	} else {
		fmt.Printf("✗ %s\n", message)
	}
}

func DisplaySuccess(message string, usePrettyOutput bool) {
	if usePrettyOutput {
		successIcon := successStyle.Render("✓")
		styledMessage := successStyle.Render(message)
		fmt.Printf("%s %s\n", successIcon, styledMessage)
	} else {
		fmt.Printf("✓ %s\n", message)
	}
}

func DisplayInfo(message string, usePrettyOutput bool) {
	if usePrettyOutput {
		styledMessage := infoStyle.Render(message)
		fmt.Println(styledMessage)
	} else {
		fmt.Println(message)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

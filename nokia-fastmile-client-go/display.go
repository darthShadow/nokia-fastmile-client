package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type MemoryInfo struct {
	TotalMB     float64
	UsedMB      float64
	FreeMB      float64
	UsedPercent float64
}

func RenderStatusBoxLipglossWithType(status *DeviceStatus, gatewayType, gatewayIP string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")). // Blue for titles
		Padding(0, 0).
		Align(lipgloss.Center)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")). // Muted cyan for borders - complements the blue titles
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")). // Light gray for labels - readable but subtle
		Width(8).
		Align(lipgloss.Left)

	deviceValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true) // Orange for model/serial
	versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("135")).Bold(true)     // Purple for version
	uptimeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("78")).Bold(true)       // Lime green for uptime

	title := titleStyle.Render(fmt.Sprintf("Nokia FastMile 5G Gateway (%s)", gatewayType))
	subtitle := titleStyle.Render(fmt.Sprintf("IP: %s", gatewayIP))

	version := status.SoftwareVersion
	if len(version) > 40 {
		version = version[:37] + "..."
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return labelStyle
			}
			switch row {
			case 0, 1:
				return deviceValueStyle
			case 2:
				return versionStyle
			case 3:
				return uptimeStyle
			default:
				return lipgloss.NewStyle()
			}
		}).
		Rows(
			[]string{"Model:", status.ModelName},
			[]string{"Serial:", status.SerialNumber},
			[]string{"Version:", version},
			[]string{"Uptime:", FormatUptime(status.UpTime)},
		)

	deviceInfo := t.Render()

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")). // Same muted cyan as borders - visual consistency
		Render(strings.Repeat("─", 58))

	cpuUsage := status.CPUUsageInfo.CPUUsage
	memInfo := FormatMemory(status.MemInfo.Total, status.MemInfo.Free)

	performanceBars := renderPerformanceBars(cpuUsage, memInfo, 58)

	content := lipgloss.JoinVertical(lipgloss.Center, title, subtitle, deviceInfo, separator, performanceBars)

	return boxStyle.Render(content) + "\n"
}

func RenderStatusBoxLipgloss(status *DeviceStatus) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(1, 0).
		Align(lipgloss.Center)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")). // slightly muted white
		Width(8).
		Align(lipgloss.Left)

	deviceValueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("87")).Bold(true) // bright cyan
	versionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))                // light blue
	uptimeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("120"))                // light green

	title := titleStyle.Render("Nokia FastMile 5G Gateway")

	version := status.SoftwareVersion
	if len(version) > 40 {
		version = version[:37] + "..."
	}

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if col == 0 {
				return labelStyle
			}
			switch row {
			case 0, 1: // Model, Serial
				return deviceValueStyle
			case 2: // Version
				return versionStyle
			case 3: // Uptime
				return uptimeStyle
			default:
				return deviceValueStyle
			}
		}).
		Rows([][]string{
			{"Model:  ", status.ModelName},
			{"Serial: ", status.SerialNumber},
			{"Version:", version},
			{"Uptime: ", FormatUptime(status.UpTime)},
		}...)

	deviceInfo := t.String()

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Render(strings.Repeat("─", 58))

	cpuUsage := status.CPUUsageInfo.CPUUsage
	memInfo := FormatMemory(status.MemInfo.Total, status.MemInfo.Free)

	performanceBars := renderPerformanceBars(cpuUsage, memInfo, 58)

	content := lipgloss.JoinVertical(lipgloss.Left,
		deviceInfo,
		separator,
		performanceBars,
	)

	boxed := boxStyle.Render(content)

	return lipgloss.JoinVertical(lipgloss.Center, title, boxed) + "\n"
}

func renderPerformanceBars(cpuUsage int, memInfo MemoryInfo, totalWidth int) string {
	barWidth := 25

	cpuFilled := max(min(int(float64(cpuUsage)/100*float64(barWidth)), barWidth), 0)

	memFilled := max(min(int(memInfo.UsedPercent/100*float64(barWidth)), barWidth), 0)

	// Consistent muted color scheme for progress bars
	getBarColor := func(percentage float64) lipgloss.Color {
		if percentage < 25 {
			return lipgloss.Color("10") // Muted green - low usage, good
		} else if percentage < 50 {
			return lipgloss.Color("11") // Muted yellow - moderate usage
		} else if percentage < 75 {
			return lipgloss.Color("3") // Muted orange - high usage, caution
		} else {
			return lipgloss.Color("9") // Muted red - very high usage, warning
		}
	}

	cpuColor := getBarColor(float64(cpuUsage))
	memColor := getBarColor(memInfo.UsedPercent)

	filledStyle := lipgloss.NewStyle()
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	cpuBar := filledStyle.Copy().Foreground(cpuColor).Render(strings.Repeat("█", cpuFilled)) +
		emptyStyle.Render(strings.Repeat("░", barWidth-cpuFilled))

	memBar := filledStyle.Copy().Foreground(memColor).Render(strings.Repeat("█", memFilled)) +
		emptyStyle.Render(strings.Repeat("░", barWidth-memFilled))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")). // Light gray for performance labels - readable but subtle
		Width(8).
		Align(lipgloss.Left)

	cpuValueStyle := lipgloss.NewStyle().
		Foreground(cpuColor). // Match CPU bar color
		Width(18).
		Align(lipgloss.Right).
		PaddingRight(1).
		Bold(true)

	memValueStyle := lipgloss.NewStyle().
		Foreground(memColor). // Match Memory bar color
		Width(18).
		Align(lipgloss.Right).
		PaddingRight(1).
		Bold(true)

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Width(totalWidth).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
				return labelStyle
			case 1:
				if row == 0 {
					return cpuValueStyle
				}
				return memValueStyle
			default:
				return lipgloss.NewStyle()
			}
		}).
		Rows([][]string{
			{"CPU:    ", fmt.Sprintf("%d%%", cpuUsage), cpuBar},
			{"Memory: ", fmt.Sprintf("%.0f%% (%.0f/%.0fMB)", memInfo.UsedPercent, memInfo.UsedMB, memInfo.TotalMB), memBar},
		}...)

	return t.String()
}

func RenderSuccessLipgloss(message string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("40")). // Slightly brighter green - good visibility
		Bold(true)
	return style.Render("✅ " + message)
}

func RenderErrorLipgloss(message string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	return style.Render("❌ " + message)
}

func RenderInfoLipgloss(message string) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")) // Bright blue - matches Python \033[94m
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")) // Bright cyan - matches Python \033[96m
	return labelStyle.Render("🔑 Session: ") + valueStyle.Render(message)
}

func RenderTokenLipgloss(message string) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")) // Bright blue - matches Python \033[94m
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")) // Bright cyan - matches Python \033[96m
	return labelStyle.Render("🎫 Token: ") + valueStyle.Render(message)
}

func ShouldUsePlainOutput() bool {
	return os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb"
}

func RenderHeader() string {
	if ShouldUsePlainOutput() {
		return ""
	}
	return "\n🌐 Nokia FastMile 5G Gateway Client\n" + strings.Repeat("━", 35) + "\n"
}

func RenderSimpleStatus(status *DeviceStatus) string {
	var content strings.Builder

	model := status.ModelName
	if model == "" {
		model = "N/A"
	}
	serial := status.SerialNumber
	if serial == "" {
		serial = "N/A"
	}
	version := status.SoftwareVersion
	if version == "" {
		version = "N/A"
	}

	content.WriteString(fmt.Sprintf("Model: %s\n", model))
	content.WriteString(fmt.Sprintf("Serial: %s\n", serial))
	content.WriteString(fmt.Sprintf("Version: %s\n", version))
	content.WriteString(fmt.Sprintf("Uptime: %s\n", FormatUptime(status.UpTime)))
	content.WriteString(fmt.Sprintf("CPU Usage: %d%%\n", status.CPUUsageInfo.CPUUsage))

	if status.MemInfo.Total > 0 {
		memInfo := FormatMemory(status.MemInfo.Total, status.MemInfo.Free)
		content.WriteString(fmt.Sprintf("Memory: %.0f%% (%.0f/%.0fMB)\n",
			memInfo.UsedPercent, memInfo.UsedMB, memInfo.TotalMB))
	}

	return content.String()
}

func FormatUptime(seconds int) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

func FormatMemory(totalKB, freeKB int) MemoryInfo {
	usedKB := totalKB - freeKB
	var usedPercent float64
	if totalKB > 0 {
		usedPercent = float64(usedKB) / float64(totalKB) * 100
	}

	return MemoryInfo{
		TotalMB:     float64(totalKB) / 1024,
		UsedMB:      float64(usedKB) / 1024,
		FreeMB:      float64(freeKB) / 1024,
		UsedPercent: usedPercent,
	}
}

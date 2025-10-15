package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// ============================================================================
// Table Styles
// ============================================================================

var (
	// Table header style
	styleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Align(lipgloss.Center)

	// Table cell style
	styleTableCell = lipgloss.NewStyle().
			Padding(0, 1)

	// Table border style
	styleTableBorder = lipgloss.NewStyle().
				Foreground(ColorTextDim)
)

// ============================================================================
// Service Table - For 'ork ps' command
// ============================================================================

// ServiceRow represents a single row in the service table
type ServiceRow struct {
	Service     string
	Status      string
	Ports       []string
	ContainerID string
	Uptime      string
}

// ServiceTable creates and renders a beautiful table for services
func ServiceTable(projectName string, rows []ServiceRow) string {
	if len(rows) == 0 {
		return renderEmptyState(projectName)
	}

	// Create a table with headers
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styleTableBorder).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Header row
			if row == 0 {
				return styleTableHeader
			}
			// Regular cells
			return styleTableCell
		}).
		Headers("SERVICE", "STATUS", "PORTS", "UPTIME", "CONTAINER")

	// Add rows
	for _, r := range rows {
		ports := formatPorts(r.Ports)
		uptime := r.Uptime
		if uptime == "" {
			uptime = Dim("-")
		}

		// Format container ID (short version)
		containerID := r.ContainerID
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}
		containerID = Dim(containerID)

		t.Row(
			r.Service,
			FormatServiceStatus(r.Status),
			ports,
			uptime,
			containerID,
		)
	}

	// Build output with a header
	var output strings.Builder
	headerText := StyleSubheader.Render(fmt.Sprintf("%s Services for project: %s", SymbolPackage, Bold(projectName)))
	output.WriteString(headerText)
	output.WriteString("\n\n")
	output.WriteString(t.String())
	output.WriteString("\n")

	return output.String()
}

// ============================================================================
// Port Table - For 'ork ports' command (future)
// ============================================================================

// PortRow represents a single row in the port allocation table
type PortRow struct {
	Port    string
	Service string
	Project string
	Host    string
}

// PortTable creates and renders a table for port allocations
func PortTable(rows []PortRow) string {
	if len(rows) == 0 {
		return renderNoPortsAllocated()
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styleTableBorder).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return styleTableHeader
			}
			return styleTableCell
		}).
		Headers("PORT", "SERVICE", "PROJECT", "HOST")

	for _, r := range rows {
		t.Row(
			Bold(r.Port),
			r.Service,
			r.Project,
			Dim(r.Host),
		)
	}

	var output strings.Builder
	headerText := StyleSubheader.Render(fmt.Sprintf("%s Port Allocations", SymbolGear))
	output.WriteString(headerText)
	output.WriteString("\n\n")
	output.WriteString(t.String())
	output.WriteString("\n")

	return output.String()
}

// ============================================================================
// Key-Value Table - For displaying configuration
// ============================================================================

// KeyValueRow represents a key-value pair
type KeyValueRow struct {
	Key   string
	Value string
}

// KeyValueTable creates a simple two-column table for configuration display
func KeyValueTable(title string, rows []KeyValueRow) string {
	if len(rows) == 0 {
		return ""
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styleTableBorder).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return styleTableHeader
			}
			// Bold the keys
			if col == 0 {
				return styleTableCell.Bold(true).Foreground(ColorSecondary)
			}
			return styleTableCell
		}).
		Headers("KEY", "VALUE")

	for _, r := range rows {
		t.Row(r.Key, r.Value)
	}

	var output strings.Builder
	if title != "" {
		headerText := StyleSubheader.Render(title)
		output.WriteString(headerText)
		output.WriteString("\n\n")
	}
	output.WriteString(t.String())
	output.WriteString("\n")

	return output.String()
}

// ============================================================================
// Health Check Table - For 'ork doctor' command
// ============================================================================

// HealthCheckRow represents a health check result
type HealthCheckRow struct {
	Check  string
	Status string // "pass", "fail", "warn"
	Detail string
}

// HealthCheckTable creates a table for health check results
func HealthCheckTable(category string, rows []HealthCheckRow) string {
	if len(rows) == 0 {
		return ""
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styleTableBorder).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return styleTableHeader
			}
			return styleTableCell
		}).
		Headers("CHECK", "STATUS", "DETAILS")

	for _, r := range rows {
		status := formatHealthStatus(r.Status)
		detail := r.Detail
		if detail == "" {
			detail = Dim("-")
		}

		t.Row(r.Check, status, detail)
	}

	var output strings.Builder
	headerText := StyleSubheader.Render(fmt.Sprintf("%s %s", SymbolDoctor, category))
	output.WriteString(headerText)
	output.WriteString("\n\n")
	output.WriteString(t.String())
	output.WriteString("\n")

	return output.String()
}

// ============================================================================
// Private Helper Functions
// ============================================================================

// formatPorts formats port list for display
func formatPorts(ports []string) string {
	if len(ports) == 0 {
		return Dim("-")
	}

	// Show the first 2 ports, add "..." if more
	if len(ports) > 2 {
		return lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Render(strings.Join(ports[:2], ", ") + "...")
	}

	return lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(strings.Join(ports, ", "))
}

// formatHealthStatus formats health check status with color
func formatHealthStatus(status string) string {
	switch status {
	case "pass", "ok":
		return StyleSuccess.Render(SymbolSuccess + " Pass")
	case "fail", "error":
		return StyleError.Render(SymbolError + " Fail")
	case "warn", "warning":
		return StyleWarning.Render(SymbolWarning + " Warn")
	default:
		return Dim(status)
	}
}

// renderEmptyState renders a message when no services are running
func renderEmptyState(projectName string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTextDim).
		Padding(1, 2).
		Align(lipgloss.Center)

	message := fmt.Sprintf(
		"%s\n\n%s\n%s %s",
		Dim("No services running"),
		fmt.Sprintf("Project: %s", Bold(projectName)),
		SymbolLightbulb,
		Dim("Start services with: "+Code("ork up")),
	)

	return box.Render(message) + "\n"
}

// renderNoPortsAllocated renders a message when no ports are allocated
func renderNoPortsAllocated() string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTextDim).
		Padding(1, 2).
		Align(lipgloss.Center)

	message := fmt.Sprintf(
		"%s\n\n%s %s",
		Dim("No ports allocated"),
		SymbolLightbulb,
		Dim("Ports are allocated when services start"),
	)

	return box.Render(message) + "\n"
}

// ============================================================================
// Simple List Renderer (alternative to tables for simpler output)
// ============================================================================

// RenderList renders a simple styled list
func RenderList(title string, items []string) string {
	var output strings.Builder

	if title != "" {
		headerText := StyleSubheader.Render(title)
		output.WriteString(headerText)
		output.WriteString("\n\n")
	}

	for _, item := range items {
		output.WriteString(fmt.Sprintf("  %s %s\n", SymbolBullet, item))
	}

	return output.String()
}

// RenderNumberedList renders a numbered list
func RenderNumberedList(title string, items []string) string {
	var output strings.Builder

	if title != "" {
		headerText := StyleSubheader.Render(title)
		output.WriteString(headerText)
		output.WriteString("\n\n")
	}

	for i, item := range items {
		number := lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true).
			Render(fmt.Sprintf("%d.", i+1))
		output.WriteString(fmt.Sprintf("  %s %s\n", number, item))
	}

	return output.String()
}

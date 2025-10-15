package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Color Scheme - Ork Brand Colors
// ============================================================================

var (
	// ColorPrimary Primary colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple - main brand
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan - accents

	// ColorSuccess Status colors
	ColorSuccess = lipgloss.Color("#10B981") // Green
	ColorWarning = lipgloss.Color("#F59E0B") // Amber
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorInfo    = lipgloss.Color("#3B82F6") // Blue

	// ColorRunning State colors
	ColorRunning  = lipgloss.Color("#10B981") // Green
	ColorStarting = lipgloss.Color("#F59E0B") // Amber
	ColorStopped  = lipgloss.Color("#6B7280") // Gray
	ColorFailed   = lipgloss.Color("#EF4444") // Red

	// ColorText Text colors
	ColorText     = lipgloss.Color("#E5E7EB") // Light gray
	ColorTextDim  = lipgloss.Color("#9CA3AF") // Dim gray
	ColorTextBold = lipgloss.Color("#F9FAFB") // Almost white

	// ColorBgDark Background colors
	ColorBgDark    = lipgloss.Color("#1F2937") // Dark gray
	ColorBgMedium  = lipgloss.Color("#374151") // Medium gray
	ColorBgLight   = lipgloss.Color("#4B5563") // Light gray
	ColorBgSuccess = lipgloss.Color("#064E3B") // Dark green
	ColorBgError   = lipgloss.Color("#7F1D1D") // Dark red
)

// ============================================================================
// Base Styles
// ============================================================================

var (
	// StyleBold Text styles
	StyleBold = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextBold)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorTextDim)

	StyleCode = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Background(ColorBgDark).
			Padding(0, 1)

	// StyleSuccess Status styles
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	StyleInfo = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	// StyleSuccessBox Box styles for callouts
	StyleSuccessBox = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(0, 1)

	StyleErrorBox = lipgloss.NewStyle().
			Foreground(ColorError).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(0, 1)

	StyleInfoBox = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorInfo).
			Padding(0, 1)

	// StyleHeader Header styles
	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Underline(true).
			MarginBottom(1)

	StyleSubheader = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)
)

// ============================================================================
// Status Indicators - Unicode symbols for terminal output
// ============================================================================

const (
	// Status symbols
	SymbolSuccess   = "✓" // Checkmark
	SymbolError     = "✗" // X mark
	SymbolWarning   = "⚠" // Warning triangle
	SymbolInfo      = "ℹ" // Info
	SymbolRunning   = "●" // Filled circle
	SymbolStarting  = "◐" // Half-filled circle
	SymbolStopped   = "○" // Empty circle
	SymbolArrow     = "→" // Right arrow
	SymbolBullet    = "•" // Bullet point
	SymbolSparkle   = "✨" // Sparkle (for special messages)
	SymbolRocket    = "🚀" // Rocket (for deployments/starts)
	SymbolPackage   = "📦" // Package (for containers)
	SymbolGear      = "⚙" // Gear (for configuration)
	SymbolDoctor    = "🩺" // Doctor (for health checks)
	SymbolLightbulb = "💡" // Lightbulb (for tips/hints)
)

// ============================================================================
// Formatted Output Functions
// ============================================================================

// Success prints a success message with a checkmark
func Success(message string) {
	fmt.Println(StyleSuccess.Render(SymbolSuccess + " " + message))
}

// Error prints an error message with X mark
func Error(message string) {
	fmt.Println(StyleError.Render(SymbolError + " " + message))
}

// Warning prints a warning message with a warning symbol
func Warning(message string) {
	fmt.Println(StyleWarning.Render(SymbolWarning + " " + message))
}

// Info prints an info message with an info symbol
func Info(message string) {
	fmt.Println(StyleInfo.Render(SymbolInfo + " " + message))
}

// Hint prints a helpful hint/tip with lightbulb
func Hint(message string) {
	fmt.Println(StyleInfo.Render(SymbolLightbulb + " " + message))
}

// Header prints a section header
func Header(message string) {
	fmt.Println(StyleHeader.Render(message))
}

// Subheader prints a subsection header
func Subheader(message string) {
	fmt.Println(StyleSubheader.Render(message))
}

// ============================================================================
// Status-Specific Formatters
// ============================================================================

// StatusRunning returns a formatted "running" status
func StatusRunning(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorRunning).
		Render(SymbolRunning + " " + text)
}

// StatusStarting returns a formatted "starting" status
func StatusStarting(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorStarting).
		Render(SymbolStarting + " " + text)
}

// StatusStopped returns a formatted "stopped" status
func StatusStopped(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorStopped).
		Render(SymbolStopped + " " + text)
}

// StatusFailed returns a formatted "failed" status
func StatusFailed(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorFailed).
		Render(SymbolError + " " + text)
}

// ============================================================================
// Service Status Formatters (for containers)
// ============================================================================

// FormatServiceStatus formats a service status with an appropriate color and symbol
func FormatServiceStatus(status string) string {
	switch status {
	case "running", "up":
		return StatusRunning("Running")
	case "starting":
		return StatusStarting("Starting")
	case "stopped", "exited":
		return StatusStopped("Stopped")
	case "failed", "error":
		return StatusFailed("Failed")
	default:
		return lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render(SymbolBullet + " " + status)
	}
}

// ============================================================================
// Inline Text Formatters (for use within strings)
// ============================================================================

// Bold returns bolded text
func Bold(text string) string {
	return StyleBold.Render(text)
}

// Dim returns dimmed text
func Dim(text string) string {
	return StyleDim.Render(text)
}

// Code returns text styled as code/command
func Code(text string) string {
	return StyleCode.Render(text)
}

// Highlight returns text in the primary brand color
func Highlight(text string) string {
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(text)
}

// ============================================================================
// Box Formatters (for callouts and important messages)
// ============================================================================

// SuccessBox prints a success message in a box
func SuccessBox(message string) {
	fmt.Println(StyleSuccessBox.Render(SymbolSuccess + " " + message))
}

// ErrorBox prints an error message in a box
func ErrorBox(message string) {
	fmt.Println(StyleErrorBox.Render(SymbolError + " " + message))
}

// InfoBox prints an info message in a box
func InfoBox(message string) {
	fmt.Println(StyleInfoBox.Render(SymbolInfo + " " + message))
}

// ============================================================================
// Utility Functions
// ============================================================================

// Separator prints a visual separator line
func Separator() {
	fmt.Println(StyleDim.Render("────────────────────────────────────────────────────────────────"))
}

// EmptyLine prints a blank line for spacing
func EmptyLine() {
	fmt.Println()
}

// List prints a bulleted list item
func List(item string) {
	fmt.Printf("  %s %s\n", StyleDim.Render(SymbolBullet), item)
}

// ListItem prints a bulleted list item with a custom prefix
func ListItem(prefix, item string) {
	fmt.Printf("  %s %s\n", prefix, item)
}

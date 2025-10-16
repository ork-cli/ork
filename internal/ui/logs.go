package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Log Level Detection
// ============================================================================

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogLevelUnknown LogLevel = iota
	LogLevelTrace
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// Common log level patterns - matches various formats:
// - ERROR, error, Error
// - [ERROR], [error]
// - level=error, level="error"
// - "level":"error"
// - ERROR:, error:
var (
	errorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(error|err|fatal|panic|critical|crit)\b`),
		regexp.MustCompile(`(?i)\[(error|err|fatal|panic|critical|crit)\]`),
		regexp.MustCompile(`(?i)level[=:]\s*"?(error|err|fatal|panic|critical|crit)"?`),
		regexp.MustCompile(`(?i)"level"\s*:\s*"(error|err|fatal|panic|critical|crit)"`),
	}

	warnPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(warn|warning|alert)\b`),
		regexp.MustCompile(`(?i)\[(warn|warning|alert)\]`),
		regexp.MustCompile(`(?i)level[=:]\s*"?(warn|warning|alert)"?`),
		regexp.MustCompile(`(?i)"level"\s*:\s*"(warn|warning|alert)"`),
	}

	infoPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(info|information|notice)\b`),
		regexp.MustCompile(`(?i)\[(info|information|notice)\]`),
		regexp.MustCompile(`(?i)level[=:]\s*"?(info|information|notice)"?`),
		regexp.MustCompile(`(?i)"level"\s*:\s*"(info|information|notice)"`),
	}

	debugPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(debug|dbg|trace|verbose)\b`),
		regexp.MustCompile(`(?i)\[(debug|dbg|trace|verbose)\]`),
		regexp.MustCompile(`(?i)level[=:]\s*"?(debug|dbg|trace|verbose)"?`),
		regexp.MustCompile(`(?i)"level"\s*:\s*"(debug|dbg|trace|verbose)"`),
	}
)

// detectLogLevel analyzes a log line and returns its detected level
func detectLogLevel(line string) LogLevel {
	// Check error patterns first (highest priority)
	for _, pattern := range errorPatterns {
		if pattern.MatchString(line) {
			return LogLevelError
		}
	}

	// Check warning patterns
	for _, pattern := range warnPatterns {
		if pattern.MatchString(line) {
			return LogLevelWarn
		}
	}

	// Check info patterns
	for _, pattern := range infoPatterns {
		if pattern.MatchString(line) {
			return LogLevelInfo
		}
	}

	// Check debug patterns
	for _, pattern := range debugPatterns {
		if pattern.MatchString(line) {
			return LogLevelDebug
		}
	}

	// Default to unknown
	return LogLevelUnknown
}

// ============================================================================
// Log Formatting Styles
// ============================================================================

var (
	// Log level colors
	logErrorStyle = lipgloss.NewStyle().Foreground(ColorError)
	logWarnStyle  = lipgloss.NewStyle().Foreground(ColorWarning)
	logInfoStyle  = lipgloss.NewStyle().Foreground(ColorInfo)
	logDebugStyle = lipgloss.NewStyle().Foreground(ColorTextDim)
	logTraceStyle = lipgloss.NewStyle().Foreground(ColorTextDim).Faint(true)

	// Timestamp style - dim and gray
	timestampStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Faint(true)

	// Service header styles
	serviceHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	containerIDStyle = lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Faint(true)

	streamingIndicatorStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)
)

// ============================================================================
// Log Formatters
// ============================================================================

// FormatServiceHeader formats a header for log output showing service name and container ID
func FormatServiceHeader(serviceName, containerID string, isStreaming bool) string {
	var parts []string

	// Service name in a box
	header := serviceHeaderStyle.Render(SymbolPackage + " " + serviceName)
	parts = append(parts, header)

	// Container ID (shortened to 12 chars like Docker does)
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}
	idText := containerIDStyle.Render("container: " + containerID)
	parts = append(parts, idText)

	// Streaming indicator if following
	if isStreaming {
		indicator := streamingIndicatorStyle.Render("● streaming")
		parts = append(parts, indicator)
	}

	return strings.Join(parts, "  ")
}

// FormatLogLine formats a single log line with appropriate color coding
func FormatLogLine(line string, showTimestamps bool) string {
	if line == "" {
		return ""
	}

	// Detect log level from the original line
	level := detectLogLevel(line)

	// Extract timestamp and content separately
	var styledTimestamp string
	var content string

	if showTimestamps {
		// Extract the timestamp and keep it separate
		timestamp, rest := extractTimestamp(line)
		if timestamp != "" {
			styledTimestamp = timestampStyle.Render(timestamp) + " "
			content = rest
		} else {
			content = line
		}
	} else {
		// Remove timestamps if present
		content = stripTimestamp(line)
	}

	// Apply color to content based on the log level
	var styledContent string
	switch level {
	case LogLevelError, LogLevelFatal:
		styledContent = logErrorStyle.Render(content)
	case LogLevelWarn:
		styledContent = logWarnStyle.Render(content)
	case LogLevelInfo:
		styledContent = logInfoStyle.Render(content)
	case LogLevelDebug, LogLevelTrace:
		styledContent = logDebugStyle.Render(content)
	default:
		// No special formatting for unknown level
		styledContent = content
	}

	// Combine styled timestamp with styled content
	return styledTimestamp + styledContent
}

// ============================================================================
// Timestamp Handling
// ============================================================================

// Common timestamp patterns at the start of log lines
var timestampPatterns = []*regexp.Regexp{
	// ISO 8601 timestamps: 2024-01-15T10:30:45Z, 2024-01-15T10:30:45.123Z
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z?\s+`),
	// RFC3339: 2024-01-15 10:30:45
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(\.\d+)?\s+`),
	// Docker timestamps: 2024-01-15T10:30:45.123456789Z
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s+`),
	// Short timestamps: 10:30:45
	regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(\.\d+)?\s+`),
	// Timestamps in brackets: [2024-01-15 10:30:45]
	regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(\.\d+)?\]\s+`),
}

// extractTimestamp extracts the timestamp from the beginning of a log line
// Returns the timestamp and the rest of the line separately
func extractTimestamp(line string) (timestamp string, rest string) {
	for _, pattern := range timestampPatterns {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) > 0 {
			timestamp = strings.TrimSpace(matches[0])
			rest = strings.TrimPrefix(line, matches[0])
			return timestamp, rest
		}
	}
	return "", line
}

// stripTimestamp removes timestamps from the beginning of log lines
func stripTimestamp(line string) string {
	for _, pattern := range timestampPatterns {
		if pattern.MatchString(line) {
			return pattern.ReplaceAllString(line, "")
		}
	}
	return line
}

// ============================================================================
// Stream State Indicator
// ============================================================================

// FormatStreamingFooter shows a footer when streaming is active
func FormatStreamingFooter() string {
	return StyleDim.Render("\n" + SymbolInfo + " Press Ctrl+C to stop streaming")
}

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Simple Spinner - No bubble tea, just clean terminal output
// ============================================================================

// Spinner represents a simple terminal spinner for indicating progress
type Spinner struct {
	message    string
	frames     []string
	frameIndex int
	isRunning  bool
	done       chan bool
	mu         sync.Mutex
	style      lipgloss.Style
}

// Default spinner frames (dots)
var defaultFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

// ============================================================================
// Constructor
// ============================================================================

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:    message,
		frames:     defaultFrames,
		frameIndex: 0,
		isRunning:  false,
		done:       make(chan bool),
		style: lipgloss.NewStyle().
			Foreground(ColorSecondary),
	}
}

// ============================================================================
// Lifecycle Methods
// ============================================================================

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	go s.run()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	s.mu.Unlock()

	s.done <- true
	s.clearLine()
}

// Success stops the spinner and shows a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	Success(message)
}

// Error stops the spinner and shows an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	Error(message)
}

// Warning stops the spinner and shows a warning message
func (s *Spinner) Warning(message string) {
	s.Stop()
	Warning(message)
}

// UpdateMessage changes the spinner message while it's running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// ============================================================================
// Private Methods
// ============================================================================

// run is the main spinner loop
func (s *Spinner) run() {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.render()
			s.nextFrame()
		}
	}
}

// render draws the current spinner frame
func (s *Spinner) render() {
	s.mu.Lock()
	frame := s.frames[s.frameIndex]
	message := s.message
	s.mu.Unlock()

	// Clear line and print spinner and message
	fmt.Printf("\r%s %s", s.style.Render(frame), message)
}

// clearLine clears the current line
func (s *Spinner) clearLine() {
	fmt.Print("\r\033[K") // ANSI escape: clear line
}

// nextFrame advances to the next spinner frame
func (s *Spinner) nextFrame() {
	s.mu.Lock()
	s.frameIndex = (s.frameIndex + 1) % len(s.frames)
	s.mu.Unlock()
}

// ============================================================================
// Alternative Spinner Styles
// ============================================================================

// WithDots creates a spinner with dot frames (default)
func (s *Spinner) WithDots() *Spinner {
	s.frames = defaultFrames
	return s
}

// WithLine creates a spinner with line frames
func (s *Spinner) WithLine() *Spinner {
	s.frames = []string{"-", "\\", "|", "/"}
	return s
}

// WithArrow creates a spinner with arrow frames
func (s *Spinner) WithArrow() *Spinner {
	s.frames = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	return s
}

// WithCircle creates a spinner with circle frames
func (s *Spinner) WithCircle() *Spinner {
	s.frames = []string{"◐", "◓", "◑", "◒"}
	return s
}

// WithColor sets a custom color for the spinner
func (s *Spinner) WithColor(color lipgloss.Color) *Spinner {
	s.style = s.style.Foreground(color)
	return s
}

// ============================================================================
// Multi-Step Progress Tracker
// ============================================================================

// ProgressTracker tracks progress through multiple steps
type ProgressTracker struct {
	steps       []string
	currentStep int
	totalSteps  int
	mu          sync.Mutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(steps []string) *ProgressTracker {
	return &ProgressTracker{
		steps:       steps,
		currentStep: 0,
		totalSteps:  len(steps),
	}
}

// Start prints the initial progress state
func (p *ProgressTracker) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	Header(fmt.Sprintf("Starting %d steps", p.totalSteps))
	EmptyLine()
}

// NextStep advances to the next step and prints progress
func (p *ProgressTracker) NextStep() {
	p.mu.Lock()
	if p.currentStep > 0 {
		// Mark the previous step as done
		prevStep := p.steps[p.currentStep-1]
		Success(prevStep)
	}

	if p.currentStep < p.totalSteps {
		// Show the current step
		currentStep := p.steps[p.currentStep]
		progress := fmt.Sprintf("[%d/%d]", p.currentStep+1, p.totalSteps)
		fmt.Printf("%s %s %s\n",
			StyleDim.Render(progress),
			SymbolArrow,
			Bold(currentStep),
		)
	}

	p.currentStep++
	p.mu.Unlock()
}

// Complete marks all steps as complete
func (p *ProgressTracker) Complete(message string) {
	p.mu.Lock()
	if p.currentStep == p.totalSteps {
		// Mark the last step as done
		lastStep := p.steps[p.totalSteps-1]
		Success(lastStep)
	}
	p.mu.Unlock()

	EmptyLine()
	SuccessBox(message)
}

// Fail marks the progress as failed
func (p *ProgressTracker) Fail(message string) {
	p.mu.Lock()
	currentStep := p.steps[p.currentStep-1]
	p.mu.Unlock()

	Error(fmt.Sprintf("%s failed", currentStep))
	EmptyLine()
	ErrorBox(message)
}

// ============================================================================
// Convenience Functions
// ============================================================================

// ShowSpinner creates, starts, and returns a spinner (convenience function)
func ShowSpinner(message string) *Spinner {
	spinner := NewSpinner(message)
	spinner.Start()
	return spinner
}

// WithProgress wraps a long-running operation with a spinner
func WithProgress(message string, fn func() error) error {
	spinner := ShowSpinner(message)
	err := fn()
	if err != nil {
		spinner.Error(message + " failed")
		return err
	}
	spinner.Success(message)
	return nil
}

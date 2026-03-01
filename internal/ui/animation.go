package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Animation timing constants - tweak these to adjust feel
const (
	AnimationTickRate  = 16 * time.Millisecond // ~60fps
	PulsePeriodSeconds = 2.5                   // Full breathing cycle
)

// Lock-in flash timing
const (
	LockInFlashDuration      = 48 * time.Millisecond // 3 frames at 60fps for snappy feedback
	FlashVisibilityThreshold = 0.3                   // Minimum intensity (30%) to show flash color
)

// GradientColors is the signature palette: light green → dark blue
var GradientColors = []string{
	"#90EE90", "#8BE88C", "#86E288", "#81DC88", "#7CD688",
	"#77D088", "#72CA88", "#6DC488", "#68BE88", "#63B888",
	"#5EB288", "#59AC88", "#54A688", "#4FA088", "#4A9A88",
	"#459488", "#408E88", "#3B8888", "#368288", "#317C88",
	"#2C7688", "#277088", "#226A88", "#1D6488", "#185E88",
	"#135888", "#0E5288", "#094C88", "#044688", "#004088",
}

// Gradient indices for different pulse states
// Gradient goes from light green (0) -> dark blue (29)
const (
	GradientDarkestIdx   = 27 // Dark blue-green for pulse valley
	GradientBrightestIdx = 3  // Light green for pulse peak
)

// AnimationState tracks the current animation state for the breathing pulse effect.
// The PulsePhase cycles from 0.0 (darkest) to 1.0 (brightest) using a sine wave
// to create a smooth breathing animation on focused UI elements.
type AnimationState struct {
	PulsePhase float64   // 0-1, sine wave phase (0=darkest valley, 1=brightest peak)
	StartTime  time.Time // When animation started, used to calculate elapsed time

	// Lock-in flash state - provides satisfying "button press" feedback on selection
	LockInActive    bool        // True when flash is currently visible
	LockInIntensity float64     // 1.0 (full bright) → 0.0 (normal), decays linearly
	LockInStartTime time.Time   // When flash started, used to calculate decay
	LockInTarget    FocusColumn // Which column triggered the flash
}

// animationTickMsg is sent periodically to update animations
type animationTickMsg struct {
	Time time.Time
}

// animationTickCmd returns a command that sends animationTickMsg every AnimationTickRate
func animationTickCmd() tea.Cmd {
	return tea.Tick(AnimationTickRate, func(t time.Time) tea.Msg {
		return animationTickMsg{Time: t}
	})
}

// lockInMsg is sent when a selection is locked in (Enter pressed)
type lockInMsg struct {
	Column FocusColumn
}

// lockInCmd creates a command that triggers a lock-in flash effect for the given column
func lockInCmd(column FocusColumn) tea.Cmd {
	return func() tea.Msg {
		return lockInMsg{Column: column}
	}
}

// FlashColor is the bright color used for lock-in feedback (vibrant cyan for contrast with gradient)
var FlashColor = lipgloss.Color("51") // Bright cyan

// getFlashIntensity returns the current flash intensity (0.0-1.0) for rendering
// Returns 0 if no flash is active
func (a AnimationState) getFlashIntensity() float64 {
	if !a.LockInActive {
		return 0
	}
	return a.LockInIntensity
}

// shouldShowFlash returns true if flash should be rendered for given column
func (a AnimationState) shouldShowFlash(column FocusColumn) bool {
	return a.LockInActive && a.LockInTarget == column && a.LockInIntensity > FlashVisibilityThreshold
}

// getPulsingColor returns a color from the gradient based on pulse phase
// phase: 0.0 = darkest (valley), 0.5 = base, 1.0 = brightest (peak)
func getPulsingColor(phase float64) lipgloss.Color {
	// Map phase 0-1 to gradient index range
	// GradientDarkestIdx (27) is darkest, GradientBrightestIdx (3) is brightest
	// So as phase increases, we move toward lower indices (lighter colors)
	idx := GradientDarkestIdx - int(phase*float64(GradientDarkestIdx-GradientBrightestIdx))

	// Clamp to valid range
	if idx < GradientBrightestIdx {
		idx = GradientBrightestIdx
	}
	if idx > GradientDarkestIdx {
		idx = GradientDarkestIdx
	}

	return lipgloss.Color(GradientColors[idx])
}

// newGradientDelegate creates a list delegate with gradient-colored selected items
func newGradientDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()

	// Use a mid-range gradient color for selected items (base state)
	selectedColor := lipgloss.Color(GradientColors[12]) // Mid green-blue

	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(selectedColor).
		BorderLeftForeground(selectedColor)

	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(selectedColor).
		BorderLeftForeground(selectedColor)

	return delegate
}

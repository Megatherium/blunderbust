package ui

import (
	"math"
	"testing"
	"time"
)

func TestAnimationStateInitialization(t *testing.T) {
	state := AnimationState{
		StartTime: time.Now(),
	}

	if state.StartTime.IsZero() {
		t.Error("AnimationState.StartTime should be set")
	}

	if state.PulsePhase != 0 {
		t.Errorf("AnimationState.PulsePhase should start at 0, got %f", state.PulsePhase)
	}
}

func TestGetPulsingColor(t *testing.T) {
	// Test at phase 0 (should return darkest color)
	color0 := getPulsingColor(0.0)
	if color0 == "" {
		t.Error("getPulsingColor(0.0) should return a color")
	}

	// Test at phase 0.5 (should return mid-range color)
	color05 := getPulsingColor(0.5)
	if color05 == "" {
		t.Error("getPulsingColor(0.5) should return a color")
	}

	// Test at phase 1.0 (should return brightest color)
	color1 := getPulsingColor(1.0)
	if color1 == "" {
		t.Error("getPulsingColor(1.0) should return a color")
	}

	// Verify they're different colors
	if color0 == color1 {
		t.Error("getPulsingColor(0.0) and getPulsingColor(1.0) should return different colors")
	}
}

func TestGetPulsingColorEdgeCases(t *testing.T) {
	// Test out-of-bounds phase values are clamped correctly
	// Phase < 0 should clamp to darkest
	colorNegative := getPulsingColor(-0.5)
	colorZero := getPulsingColor(0.0)
	if colorNegative != colorZero {
		t.Error("getPulsingColor(-0.5) should clamp to same color as phase 0.0")
	}

	// Phase > 1 should clamp to brightest
	colorOverOne := getPulsingColor(1.5)
	colorOne := getPulsingColor(1.0)
	if colorOverOne != colorOne {
		t.Error("getPulsingColor(1.5) should clamp to same color as phase 1.0")
	}
}

func TestGetPulsingColorUsesGradient(t *testing.T) {
	// Verify the function uses our gradient colors
	color := getPulsingColor(0.5)
	colorStr := string(color)

	// Check it's one of our gradient colors
	found := false
	for _, gradientColor := range GradientColors {
		if colorStr == gradientColor {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("getPulsingColor should return a color from GradientColors, got %s", colorStr)
	}
}

func TestGradientConstants(t *testing.T) {
	// Verify gradient indices are within bounds
	if GradientDarkestIdx < 0 || GradientDarkestIdx >= len(GradientColors) {
		t.Errorf("GradientDarkestIdx (%d) is out of bounds [0, %d)", GradientDarkestIdx, len(GradientColors))
	}

	if GradientBrightestIdx < 0 || GradientBrightestIdx >= len(GradientColors) {
		t.Errorf("GradientBrightestIdx (%d) is out of bounds [0, %d)", GradientBrightestIdx, len(GradientColors))
	}

	// Darkest should be at a higher index than brightest (gradient goes light -> dark)
	if GradientDarkestIdx <= GradientBrightestIdx {
		t.Errorf("GradientDarkestIdx (%d) should be greater than GradientBrightestIdx (%d)", GradientDarkestIdx, GradientBrightestIdx)
	}
}

func TestAnimationConstants(t *testing.T) {
	// Verify animation timing constants are reasonable
	if AnimationTickRate <= 0 {
		t.Error("AnimationTickRate should be positive")
	}

	if PulsePeriodSeconds <= 0 {
		t.Error("PulsePeriodSeconds should be positive")
	}

	// Tick rate should be much smaller than period
	periodDuration := time.Duration(PulsePeriodSeconds * float64(time.Second))
	if AnimationTickRate >= periodDuration {
		t.Error("AnimationTickRate should be much smaller than PulsePeriodSeconds")
	}
}

func TestAnimationTickCmd(t *testing.T) {
	cmd := animationTickCmd()
	if cmd == nil {
		t.Error("animationTickCmd() should return a non-nil command")
	}
}

func TestHandleAnimationTick(t *testing.T) {
	// Create a model with initialized animation state
	m := UIModel{
		animState: AnimationState{
			StartTime:  time.Now().Add(-time.Second), // Started 1 second ago
			PulsePhase: 0.5,
		},
	}

	msg := animationTickMsg{Time: time.Now()}

	newModel, cmd := m.handleAnimationTick(msg)
	newM := newModel.(UIModel)

	// Phase should have changed
	if newM.animState.PulsePhase == m.animState.PulsePhase {
		t.Error("handleAnimationTick should update PulsePhase")
	}

	// Should return a command to continue the animation loop
	if cmd == nil {
		t.Error("handleAnimationTick should return a command to continue animation")
	}

	// Phase should be in valid range [0, 1]
	if newM.animState.PulsePhase < 0 || newM.animState.PulsePhase > 1 {
		t.Errorf("PulsePhase should be in range [0, 1], got %f", newM.animState.PulsePhase)
	}
}

func TestPulsePhaseCalculation(t *testing.T) {
	// Test the sine wave calculation directly
	startTime := time.Now()

	testCases := []struct {
		elapsed  float64 // seconds
		expected float64 // approximate expected phase
	}{
		{0.0, 0.5},                        // Start of cycle, sin(0) = 0, normalized = 0.5
		{PulsePeriodSeconds / 4, 1.0},     // Quarter period, sin(π/2) = 1, normalized = 1.0
		{PulsePeriodSeconds / 2, 0.5},     // Half period, sin(π) = 0, normalized = 0.5
		{PulsePeriodSeconds * 3 / 4, 0.0}, // Three-quarter period, sin(3π/2) = -1, normalized = 0.0
		{PulsePeriodSeconds, 0.5},         // Full period, sin(2π) = 0, normalized = 0.5
	}

	for _, tc := range testCases {
		elapsed := tc.elapsed
		period := PulsePeriodSeconds
		phase := (math.Sin(2*math.Pi*elapsed/period) + 1) / 2

		// Allow some tolerance for floating point math
		tolerance := 0.01
		if math.Abs(phase-tc.expected) > tolerance {
			t.Errorf("Pulse phase at %f seconds: expected %f, got %f", tc.elapsed, tc.expected, phase)
		}
	}

	_ = startTime // Silence unused variable warning if we ever change the test
}

func TestAnimationIntegrationInUIModel(t *testing.T) {
	// Verify that UIModel has the animState field and it's accessible
	m := UIModel{}

	// Should be able to set animation state
	m.animState = AnimationState{
		StartTime:  time.Now(),
		PulsePhase: 0.75,
	}

	if m.animState.PulsePhase != 0.75 {
		t.Errorf("Failed to set PulsePhase, expected 0.75, got %f", m.animState.PulsePhase)
	}
}

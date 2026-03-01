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

func TestLockInFlashConstants(t *testing.T) {
	// Verify lock-in flash timing constant is reasonable
	if LockInFlashDuration <= 0 {
		t.Error("LockInFlashDuration should be positive")
	}

	// Flash should be very brief (less than 100ms for snappy feedback)
	if LockInFlashDuration > 100*time.Millisecond {
		t.Errorf("LockInFlashDuration (%v) should be <= 100ms for snappy feedback", LockInFlashDuration)
	}

	// Flash should be noticeable (at least 16ms = 1 frame at 60fps)
	if LockInFlashDuration < 16*time.Millisecond {
		t.Errorf("LockInFlashDuration (%v) should be >= 16ms to be visible", LockInFlashDuration)
	}
}

func TestLockInFlashState(t *testing.T) {
	state := AnimationState{
		StartTime:       time.Now(),
		LockInActive:    true,
		LockInIntensity: 1.0,
		LockInStartTime: time.Now(),
		LockInTarget:    FocusTickets,
	}

	// Verify initial state
	if !state.LockInActive {
		t.Error("LockInActive should be true")
	}

	if state.LockInIntensity != 1.0 {
		t.Errorf("LockInIntensity should start at 1.0, got %f", state.LockInIntensity)
	}

	if state.LockInTarget != FocusTickets {
		t.Errorf("LockInTarget should be FocusTickets, got %v", state.LockInTarget)
	}
}

func TestGetFlashIntensity(t *testing.T) {
	// Test when flash is inactive
	inactiveState := AnimationState{
		LockInActive:    false,
		LockInIntensity: 0.0,
	}

	if inactiveState.getFlashIntensity() != 0 {
		t.Error("getFlashIntensity() should return 0 when flash is inactive")
	}

	// Test when flash is active
	activeState := AnimationState{
		LockInActive:    true,
		LockInIntensity: 0.75,
	}

	if activeState.getFlashIntensity() != 0.75 {
		t.Errorf("getFlashIntensity() should return %f, got %f", 0.75, activeState.getFlashIntensity())
	}
}

func TestShouldShowFlash(t *testing.T) {
	// Test when flash should be shown (active, correct column, high intensity)
	state := AnimationState{
		LockInActive:    true,
		LockInIntensity: 0.5,
		LockInTarget:    FocusTickets,
	}

	if !state.shouldShowFlash(FocusTickets) {
		t.Error("shouldShowFlash should return true for matching column with intensity > 0.3")
	}

	// Test wrong column
	if state.shouldShowFlash(FocusHarness) {
		t.Error("shouldShowFlash should return false for non-matching column")
	}

	// Test low intensity (below threshold)
	state.LockInIntensity = 0.2
	if state.shouldShowFlash(FocusTickets) {
		t.Error("shouldShowFlash should return false when intensity is below 0.3")
	}

	// Test inactive flash
	state.LockInActive = false
	state.LockInIntensity = 1.0
	if state.shouldShowFlash(FocusTickets) {
		t.Error("shouldShowFlash should return false when flash is inactive")
	}
}

func TestLockInCmd(t *testing.T) {
	cmd := lockInCmd(FocusTickets)
	if cmd == nil {
		t.Error("lockInCmd() should return a non-nil command")
	}

	// Execute the command and verify it returns the correct message
	msg := cmd()
	lockIn, ok := msg.(lockInMsg)
	if !ok {
		t.Errorf("lockInCmd() should return a lockInMsg, got %T", msg)
	}

	if lockIn.Column != FocusTickets {
		t.Errorf("lockInMsg.Column should be FocusTickets, got %v", lockIn.Column)
	}
}

func TestHandleAnimationTickWithLockInFlash(t *testing.T) {
	startTime := time.Now()

	// Create a model with active lock-in flash
	m := UIModel{
		animState: AnimationState{
			StartTime:       startTime,
			PulsePhase:      0.5,
			LockInActive:    true,
			LockInIntensity: 1.0,
			LockInStartTime: startTime,
			LockInTarget:    FocusTickets,
		},
	}

	// Test immediately after flash starts (intensity should still be high)
	msg := animationTickMsg{Time: startTime.Add(10 * time.Millisecond)}
	newModel, _ := m.handleAnimationTick(msg)
	newM := newModel.(UIModel)

	if !newM.animState.LockInActive {
		t.Error("LockInActive should still be true shortly after flash starts")
	}

	if newM.animState.LockInIntensity >= 1.0 || newM.animState.LockInIntensity <= 0 {
		t.Errorf("LockInIntensity should have decayed from 1.0, got %f", newM.animState.LockInIntensity)
	}

	// Test after flash duration (should be complete)
	msg2 := animationTickMsg{Time: startTime.Add(LockInFlashDuration + 10*time.Millisecond)}
	finalModel, _ := newM.handleAnimationTick(msg2)
	finalM := finalModel.(UIModel)

	if finalM.animState.LockInActive {
		t.Error("LockInActive should be false after flash duration")
	}

	if finalM.animState.LockInIntensity != 0.0 {
		t.Errorf("LockInIntensity should be 0.0 after flash complete, got %f", finalM.animState.LockInIntensity)
	}
}

func TestLockInFlashDecayCalculation(t *testing.T) {
	startTime := time.Now()

	testCases := []struct {
		elapsedMs      int64
		expectedActive bool
		minIntensity   float64
		maxIntensity   float64
	}{
		{0, true, 0.99, 1.0},   // Start: full intensity
		{24, true, 0.4, 0.6},   // Halfway: ~50% intensity
		{47, true, 0.0, 0.1},   // Near end: very low intensity
		{48, false, 0.0, 0.0},  // Exactly at duration: should be complete
		{100, false, 0.0, 0.0}, // After duration: definitely complete
	}

	for _, tc := range testCases {
		m := UIModel{
			animState: AnimationState{
				StartTime:       startTime,
				LockInActive:    true,
				LockInIntensity: 1.0,
				LockInStartTime: startTime,
				LockInTarget:    FocusTickets,
			},
		}

		msg := animationTickMsg{Time: startTime.Add(time.Duration(tc.elapsedMs) * time.Millisecond)}
		newModel, _ := m.handleAnimationTick(msg)
		newM := newModel.(UIModel)

		if newM.animState.LockInActive != tc.expectedActive {
			t.Errorf("At %dms: expected LockInActive=%v, got %v", tc.elapsedMs, tc.expectedActive, newM.animState.LockInActive)
		}

		if newM.animState.LockInIntensity < tc.minIntensity || newM.animState.LockInIntensity > tc.maxIntensity {
			t.Errorf("At %dms: expected intensity in [%f, %f], got %f", tc.elapsedMs, tc.minIntensity, tc.maxIntensity, newM.animState.LockInIntensity)
		}
	}
}

func TestLockInMsgHandler(t *testing.T) {
	// This tests the message structure that's handled in model.go
	// Verify the message can be created with the correct column
	msg := lockInMsg{Column: FocusHarness}

	if msg.Column != FocusHarness {
		t.Errorf("lockInMsg.Column should be FocusHarness, got %v", msg.Column)
	}

	// Verify other focus columns work too
	msg2 := lockInMsg{Column: FocusTickets}
	if msg2.Column != FocusTickets {
		t.Errorf("lockInMsg.Column should be FocusTickets, got %v", msg2.Column)
	}
}

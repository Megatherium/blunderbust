package ui

import (
	"math"
	"testing"
	"time"
)

func TestHandleAnimationTick_PulsePhaseCalculation(t *testing.T) {
	tests := []struct {
		name           string
		elapsedSeconds float64
		expectedPhase  float64
	}{
		{
			name:           "zero elapsed time",
			elapsedSeconds: 0.0,
			expectedPhase:  0.5, // sin(0) = 0, normalized = 0.5
		},
		{
			name:           "quarter period",
			elapsedSeconds: PulsePeriodSeconds / 4,
			expectedPhase:  1.0, // sin(π/2) = 1, normalized = 1.0
		},
		{
			name:           "half period",
			elapsedSeconds: PulsePeriodSeconds / 2,
			expectedPhase:  0.5, // sin(π) = 0, normalized = 0.5
		},
		{
			name:           "three quarter period",
			elapsedSeconds: PulsePeriodSeconds * 3 / 4,
			expectedPhase:  0.0, // sin(3π/2) = -1, normalized = 0.0
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewTestModel()
			model.animState.StartTime = time.Now()
			msg := animationTickMsg{Time: model.animState.StartTime.Add(time.Duration(tc.elapsedSeconds * float64(time.Second)))}

			newModel, _ := model.handleAnimationTick(msg)

			tolerance := 0.01
			phase := newModel.(UIModel).animState.PulsePhase
			if math.Abs(phase-tc.expectedPhase) > tolerance {
				t.Errorf("Expected phase %f, got %f (tolerance: %f)", tc.expectedPhase, phase, tolerance)
			}
		})
	}
}

func TestHandleAnimationTick_ColorCycle(t *testing.T) {
	model := NewTestModel()
	model.animState.StartTime = time.Now()
	model.animState.ColorCycleStart = time.Now()
	model.animState.ColorCycleIndex = 0
	model.currentTheme = &MatrixTheme

	// Before the color cycle interval, index should not change
	msg := animationTickMsg{Time: model.animState.ColorCycleStart.Add(ColorCycleInterval - time.Second)}
	newModel, _ := model.handleAnimationTick(msg)

	if newModel.(UIModel).animState.ColorCycleIndex != 0 {
		t.Errorf("Expected ColorCycleIndex to remain 0 before interval, got %d", newModel.(UIModel).animState.ColorCycleIndex)
	}

	// After the color cycle interval, index should increment
	msg = animationTickMsg{Time: model.animState.ColorCycleStart.Add(ColorCycleInterval)}
	newModel, _ = model.handleAnimationTick(msg)

	if newModel.(UIModel).animState.ColorCycleIndex != 1 {
		t.Errorf("Expected ColorCycleIndex to be 1 after interval, got %d", newModel.(UIModel).animState.ColorCycleIndex)
	}
}

func TestHandleAnimationTick_LockInFlashDecay(t *testing.T) {
	model := NewTestModel()
	model.animState.StartTime = time.Now()
	model.animState.LockInActive = true
	model.animState.LockInStartTime = time.Now()
	model.animState.LockInIntensity = 1.0

	// At start of flash, intensity should be 1.0
	msg := animationTickMsg{Time: model.animState.LockInStartTime}
	newModel, _ := model.handleAnimationTick(msg)

	if newModel.(UIModel).animState.LockInIntensity != 1.0 {
		t.Errorf("Expected LockInIntensity to be 1.0 at start, got %f", newModel.(UIModel).animState.LockInIntensity)
	}

	// At halfway point, intensity should be 0.5
	msg = animationTickMsg{Time: model.animState.LockInStartTime.Add(LockInFlashDuration / 2)}
	newModel, _ = model.handleAnimationTick(msg)

	if newModel.(UIModel).animState.LockInIntensity != 0.5 {
		t.Errorf("Expected LockInIntensity to be 0.5 at halfway, got %f", newModel.(UIModel).animState.LockInIntensity)
	}

	// At end of flash, should reset to inactive state
	msg = animationTickMsg{Time: model.animState.LockInStartTime.Add(LockInFlashDuration)}
	newModel, _ = model.handleAnimationTick(msg)

	if newModel.(UIModel).animState.LockInActive {
		t.Error("Expected LockInActive to be false after flash duration")
	}
	if newModel.(UIModel).animState.LockInIntensity != 0.0 {
		t.Errorf("Expected LockInIntensity to be 0.0 after flash, got %f", newModel.(UIModel).animState.LockInIntensity)
	}
}

func TestHandleAnimationTick_StopWhenIdle(t *testing.T) {
	model := NewTestModel()
	model.animState.StartTime = time.Now()
	model.animState.LockInActive = false

	msg := animationTickMsg{Time: model.animState.StartTime}
	newModel, cmd := model.handleAnimationTick(msg)

	if cmd != nil {
		t.Error("Expected command to be nil when LockInActive is false")
	}
	if newModel.(UIModel).animState.LockInActive != false {
		t.Error("Expected LockInActive to remain false")
	}
}

func TestHandleRefreshAnimationTick_FrameCycle(t *testing.T) {
	tests := []struct {
		name          string
		initialFrame  int
		expectedFrame int
	}{
		{
			name:          "cycle from 0 to 1",
			initialFrame:  0,
			expectedFrame: 1,
		},
		{
			name:          "cycle from 1 to 2",
			initialFrame:  1,
			expectedFrame: 2,
		},
		{
			name:          "cycle from 2 to 3",
			initialFrame:  2,
			expectedFrame: 3,
		},
		{
			name:          "cycle from 3 back to 0",
			initialFrame:  3,
			expectedFrame: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			model := NewTestModel()
			model.refreshAnimationFrame = tc.initialFrame
			model.refreshedRecently = true

			newModel, _ := model.handleRefreshAnimationTick()

			if newModel.(UIModel).refreshAnimationFrame != tc.expectedFrame {
				t.Errorf("Expected frame %d, got %d", tc.expectedFrame, newModel.(UIModel).refreshAnimationFrame)
			}
		})
	}
}

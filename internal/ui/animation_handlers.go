package ui

import (
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m UIModel) handleAnimationTick(msg animationTickMsg) (tea.Model, tea.Cmd) {
	elapsed := msg.Time.Sub(m.animState.StartTime).Seconds()

	// Pulse cycle: 0 to 1 to 0 over PulsePeriodSeconds
	// Using sine wave: sin(2π * t / period)
	period := PulsePeriodSeconds
	phase := (math.Sin(2*math.Pi*elapsed/period) + 1) / 2 // Normalize to 0-1

	m.animState.PulsePhase = phase

	// Handle color cycling - change palette every ColorCycleInterval
	cycleElapsed := msg.Time.Sub(m.animState.ColorCycleStart).Seconds()
	if cycleElapsed >= ColorCycleInterval.Seconds() {
		var cycleCount int
		if m.currentTheme == nil {
			cycleCount = len(MatrixThemeColorCycles)
		} else {
			switch m.currentTheme.Name {
			case CyberpunkTheme.Name:
				cycleCount = len(CyberpunkThemeColorCycles)
			case TokyoNightTheme.Name:
				cycleCount = len(TokyoNightThemeColorCycles)
			default:
				cycleCount = len(MatrixThemeColorCycles)
			}
		}
		if cycleCount < 1 {
			cycleCount = 1
		}
		m.animState.ColorCycleIndex = (m.animState.ColorCycleIndex + 1) % cycleCount
		m.animState.ColorCycleStart = msg.Time
	}

	// Decay lock-in flash intensity
	if m.animState.LockInActive {
		flashElapsed := msg.Time.Sub(m.animState.LockInStartTime).Milliseconds()
		flashDuration := int64(LockInFlashDuration / time.Millisecond)

		if flashElapsed >= flashDuration {
			// Flash complete - reset to inactive state
			m.animState.LockInActive = false
			m.animState.LockInIntensity = 0.0
		} else {
			// Linear decay: 1.0 → 0.0 over the flash duration
			m.animState.LockInIntensity = 1.0 - float64(flashElapsed)/float64(flashDuration)
		}
	}

	if !m.animState.LockInActive {
		// Stop the animation loop to conserve CPU when idle
		return m, nil
	}

	// Continue animation loop
	return m, animationTickCmd()
}

func (m UIModel) handleRefreshAnimationTick() (tea.Model, tea.Cmd) {
	if !m.refreshedRecently {
		return m, nil
	}
	m.refreshAnimationFrame = (m.refreshAnimationFrame + 1) % 4
	return m, tea.Tick(animationTickInterval, func(t time.Time) tea.Msg {
		return refreshAnimationTickMsg{}
	})
}

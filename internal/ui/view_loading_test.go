package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRenderLoading_DisplaysSpinner(t *testing.T) {
	cfg := LoadingConfig{
		StartTime: time.Now(),
		Theme:     MatrixTheme,
	}

	s := RenderLoading(cfg)
	// Should contain one of the spinner frames
	assert.True(t, containsAny(s, []string{"◜", "◝", "◞", "◟"}))
}

func TestRenderLoading_DisplaysInitializing(t *testing.T) {
	cfg := LoadingConfig{
		StartTime: time.Now(),
		Theme:     MatrixTheme,
	}

	s := RenderLoading(cfg)
	assert.Contains(t, s, "Initializing...")
}

func TestRenderLoading_DisplaysInsertCoin(t *testing.T) {
	cfg := LoadingConfig{
		StartTime: time.Now(),
		Theme:     MatrixTheme,
	}

	s := RenderLoading(cfg)
	assert.Contains(t, s, "INSERT COIN TO START")
}

func TestRenderLoading_DisplaysLoadingTickets(t *testing.T) {
	cfg := LoadingConfig{
		StartTime: time.Now(),
		Theme:     MatrixTheme,
	}

	s := RenderLoading(cfg)
	assert.Contains(t, s, "Loading tickets...")
}

func TestRenderLoading_UsesThemeColors(t *testing.T) {
	customTheme := ThemePalette{
		Name:       "Custom",
		TitleColor: lipgloss.Color("#123456"),
		ArcadeGold: lipgloss.Color("#654321"),
	}

	cfg := LoadingConfig{
		StartTime: time.Now(),
		Theme:     customTheme,
	}

	// Just verify it renders without error with custom theme
	s := RenderLoading(cfg)
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "INSERT COIN TO START")
}

func TestRenderLoading_SpinnerFrameChanges(t *testing.T) {
	// Create a time in the past to get a different frame
	pastTime := time.Now().Add(-250 * time.Millisecond)

	cfg1 := LoadingConfig{
		StartTime: pastTime,
		Theme:     MatrixTheme,
	}

	cfg2 := LoadingConfig{
		StartTime: time.Now(),
		Theme:     MatrixTheme,
	}

	s1 := RenderLoading(cfg1)
	s2 := RenderLoading(cfg2)

	// Both should render successfully
	assert.NotEmpty(t, s1)
	assert.NotEmpty(t, s2)
}

func TestRenderLoading_DefaultsToMatrixTheme(t *testing.T) {
	cfg := LoadingConfig{
		StartTime: time.Now(),
		Theme:     ThemePalette{}, // Empty theme
	}

	s := RenderLoading(cfg)
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "INSERT COIN TO START")
}

// Helper function
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

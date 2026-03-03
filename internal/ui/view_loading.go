package ui

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// LoadingConfig holds configuration for rendering the loading view
type LoadingConfig struct {
	StartTime time.Time
	Theme     *ThemePalette
}

// RenderLoading displays an arcade-style loading screen
func RenderLoading(cfg LoadingConfig) string {
	// Animated spinner frames
	frames := []string{"◜", "◝", "◞", "◟"}
	frameIndex := int(time.Since(cfg.StartTime).Seconds()*4) % 4
	frame := frames[frameIndex]

	// Use theme colors for loading
	theme := cfg.Theme
	if theme == nil {
		theme = &MatrixTheme
	}
	spinnerColor := theme.TitleColor
	arcadeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ArcadeGold)

	var s string
	s = "\n\n"
	s += lipgloss.NewStyle().Foreground(spinnerColor).Render(frame+" Initializing...") + "\n\n"
	s += arcadeStyle.Render("INSERT COIN TO START") + "\n"
	s += lipgloss.NewStyle().Faint(true).Render("(Loading tickets...)")
	return s
}

package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// MatrixConfig holds all configuration needed to render the matrix view
type MatrixConfig struct {
	Width  int
	Height int

	ShowSidebar bool

	// Column widths
	SidebarWidth int
	TWidth       int
	HWidth       int
	MWidth       int
	AWidth       int

	// Column disabled states
	ModelColumnDisabled bool
	AgentColumnDisabled bool

	// Focus state
	Focus FocusColumn

	// Animation state
	AnimState AnimationState

	// Current theme
	Theme *ThemePalette

	// List views
	TicketView  string
	HarnessView string
	ModelView   string
	AgentView   string
	SidebarView string

	// List titles for focus indicators
	TicketTitle  string
	HarnessTitle string
	ModelTitle   string
	AgentTitle   string
}

// RenderMatrix renders the main matrix view with 4 columns
func RenderMatrix(cfg MatrixConfig) string {
	// Guard against uninitialized dimensions
	if cfg.Height < filterHeight+2 {
		return "Initializing..."
	}

	listHeight := cfg.Height - filterHeight

	theme := cfg.Theme
	if theme == nil {
		theme = &MatrixTheme
	}

	activeColor := getActiveColor(cfg.AnimState, cfg.Focus, theme)
	glowColor := getGlowColor(cfg.AnimState.PulsePhase, theme)

	activeBorder := createActiveBorder(listHeight, activeColor, glowColor)
	inactiveBorder := createInactiveBorder(listHeight)

	capView := func(view string, w int) string {
		return lipgloss.NewStyle().MaxHeight(listHeight - 2).MaxWidth(w - 2).Render(view)
	}
	faintCapView := func(view string, w int) string {
		return lipgloss.NewStyle().Faint(true).MaxHeight(listHeight - 2).MaxWidth(w - 2).Render(view)
	}

	// Render columns
	tView := renderMatrixColumn(cfg.TicketView, cfg.TWidth, cfg.Focus == FocusTickets,
		cfg.TicketTitle, theme, activeBorder, inactiveBorder, capView, faintCapView)

	hView := renderMatrixColumn(cfg.HarnessView, cfg.HWidth, cfg.Focus == FocusHarness,
		cfg.HarnessTitle, theme, activeBorder, inactiveBorder, capView, faintCapView)

	mView := renderModelColumn(cfg, theme, listHeight, capView, faintCapView)
	aView := renderAgentColumn(cfg, theme, listHeight, capView, faintCapView)

	matrixWidth := cfg.TWidth + cfg.HWidth + cfg.MWidth + cfg.AWidth + 6

	filterBox := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Width(matrixWidth-2).
		Height(1).
		Padding(0, 1).
		Render("Filters: [All] | (Press / to search)")

	matrixBox := lipgloss.JoinHorizontal(lipgloss.Top,
		tView,
		lipgloss.NewStyle().Width(2).Render("  "),
		hView,
		lipgloss.NewStyle().Width(2).Render("  "),
		mView,
		lipgloss.NewStyle().Width(2).Render("  "),
		aView,
	)

	rightPanelBox := lipgloss.JoinVertical(lipgloss.Top, filterBox, matrixBox)

	if cfg.ShowSidebar {
		return renderMatrixWithSidebar(cfg, rightPanelBox, activeColor)
	}

	return rightPanelBox
}

func getActiveColor(animState AnimationState, focus FocusColumn, theme *ThemePalette) lipgloss.Color {
	if animState.shouldShowFlash(focus) {
		return FlashColor
	}
	return getCyclingColor(animState.PulsePhase, animState.ColorCycleIndex, theme)
}

func createActiveBorder(listHeight int, activeColor, glowColor lipgloss.Color) func(int) lipgloss.Style {
	return func(w int) lipgloss.Style {
		if w < 2 {
			w = 2
		}
		return lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(activeColor).
			Background(glowColor).
			Width(w - 2).
			Height(listHeight - 2)
	}
}

func createInactiveBorder(listHeight int) func(int) lipgloss.Style {
	return func(w int) lipgloss.Style {
		if w < 2 {
			w = 2
		}
		return lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(ThemeInactive).
			Faint(false).
			Width(w - 2).
			Height(listHeight - 2)
	}
}

func renderMatrixColumn(
	view string, width int,
	isFocused bool,
	title string,
	theme *ThemePalette,
	activeBorder, inactiveBorder func(int) lipgloss.Style,
	capView, faintCapView func(string, int) string,
) string {
	const focusIndicator = "▶ "
	const noIndicator = "  "

	focusedTitleStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.FocusIndicator)

	if isFocused {
		indicator := focusedTitleStyle.Render(focusIndicator)
		titledView := indicator + title + "\n" + view
		return activeBorder(width).Render(capView(titledView, width))
	}

	titledView := noIndicator + title + "\n" + view
	return inactiveBorder(width).Render(faintCapView(titledView, width))
}

func renderModelColumn(cfg MatrixConfig, theme *ThemePalette, listHeight int,
	capView, faintCapView func(string, int) string) string {
	if cfg.ModelColumnDisabled {
		disabledStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(ThemeInactive).
			Faint(true).
			Width(cfg.MWidth-2).
			Height(listHeight-2).
			Align(lipgloss.Center, lipgloss.Center)
		return disabledStyle.Render("N/A\n\nNo models available\nfor this harness")
	}

	if cfg.Focus == FocusModel {
		activeColor := getActiveColor(cfg.AnimState, FocusModel, theme)
		glowColor := getGlowColor(cfg.AnimState.PulsePhase, theme)
		activeBorder := createActiveBorder(listHeight, activeColor, glowColor)
		return activeBorder(cfg.MWidth).Render(capView(cfg.ModelView, cfg.MWidth))
	}

	inactiveBorder := createInactiveBorder(listHeight)
	return inactiveBorder(cfg.MWidth).Render(faintCapView(cfg.ModelView, cfg.MWidth))
}

func renderAgentColumn(cfg MatrixConfig, theme *ThemePalette, listHeight int,
	capView, faintCapView func(string, int) string) string {
	if cfg.AgentColumnDisabled {
		disabledStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(ThemeInactive).
			Faint(true).
			Width(cfg.AWidth-2).
			Height(listHeight-2).
			Align(lipgloss.Center, lipgloss.Center)
		return disabledStyle.Render("N/A\n\nNo agents available\nfor this harness")
	}

	if cfg.Focus == FocusAgent {
		activeColor := getActiveColor(cfg.AnimState, FocusAgent, theme)
		glowColor := getGlowColor(cfg.AnimState.PulsePhase, theme)
		activeBorder := createActiveBorder(listHeight, activeColor, glowColor)
		return activeBorder(cfg.AWidth).Render(capView(cfg.AgentView, cfg.AWidth))
	}

	inactiveBorder := createInactiveBorder(listHeight)
	return inactiveBorder(cfg.AWidth).Render(faintCapView(cfg.AgentView, cfg.AWidth))
}

func renderMatrixWithSidebar(cfg MatrixConfig, rightPanelBox string, activeColor lipgloss.Color) string {
	w := cfg.SidebarWidth
	if w < 2 {
		w = 2
	}

	sidebarBorder := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		Width(w - 2).
		Height(cfg.Height - 2)

	if cfg.Focus == FocusSidebar {
		sidebarBorder = sidebarBorder.BorderForeground(activeColor)
	} else {
		sidebarBorder = sidebarBorder.BorderForeground(ThemeInactive)
	}

	sidebarBox := sidebarBorder.Render(cfg.SidebarView)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox,
		lipgloss.NewStyle().Width(2).Render("  "), rightPanelBox)
}

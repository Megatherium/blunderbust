package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *UIModel) updateSizes() {
	if m.layout.Width == 0 || m.layout.Height == 0 {
		return
	}

	safeW := func(w int) int {
		if w-borderWidth < 1 {
			return 1
		}
		return w - borderWidth
	}

	m.ticketList.SetSize(safeW(m.layout.TWidth), m.layout.InnerListHeight)
	m.harnessList.SetSize(safeW(m.layout.HWidth), m.layout.InnerListHeight)
	m.modelList.SetSize(safeW(m.layout.MWidth), m.layout.InnerListHeight)
	m.agentList.SetSize(safeW(m.layout.AWidth), m.layout.InnerListHeight)
	m.sidebar.SetSize(m.layout.SidebarWidth, m.layout.Height)
	m.help.Width = m.layout.Width
}

func (m UIModel) getThemeValue() ThemePalette {
	if m.currentTheme != nil {
		return *m.currentTheme
	}
	return MatrixTheme
}

func (m UIModel) renderMainContent() string {
	return RenderMainContent(MainContentConfig{
		State:              m.state,
		Focus:              m.focus,
		ViewingAgentID:     m.viewingAgentID,
		Selection:          m.selection,
		Renderer:           m.app.Renderer,
		DryRun:             m.app.opts.DryRun,
		SelectedWorktree:   m.selectedWorktree,
		CurrentTheme:       m.getThemeValue(),
		ShowModal:          m.showModal,
		ModalContent:       m.modalContent,
		PendingProjectPath: m.pendingProjectPath,
		Warnings:           m.warnings,
		Width:              m.layout.Width,
		Height:             m.layout.Height,
		Err:                m.err,
		RetryStore:         m.retryStore,
		MatrixConfig:       m.buildMatrixConfig(),
		Agent:              m.agents[m.viewingAgentID],
		Filepicker:         m.filepicker,
		AnimState:          m.animState,
	})
}

func (m UIModel) buildMatrixConfig() MatrixConfig {
	var theme ThemePalette
	if m.currentTheme != nil {
		theme = *m.currentTheme
	} else {
		theme = MatrixTheme
	}
	cfg := MatrixConfig{
		Width:               m.layout.Width,
		Height:              m.layout.Height,
		ShowSidebar:         m.showSidebar,
		SidebarWidth:        m.layout.SidebarWidth,
		TWidth:              m.layout.TWidth,
		HWidth:              m.layout.HWidth,
		MWidth:              m.layout.MWidth,
		AWidth:              m.layout.AWidth,
		ModelColumnDisabled: m.modelColumnDisabled,
		AgentColumnDisabled: m.agentColumnDisabled,
		Focus:               m.focus,
		AnimState:           m.animState,
		Theme:               theme,
		TicketView:          m.ticketViewCache,
		HarnessView:         m.harnessViewCache,
		ModelView:           m.modelViewCache,
		AgentView:           m.agentViewCache,
		SidebarView:         m.sidebar.View(),
		TicketTitle:         m.ticketList.Title,
		HarnessTitle:        m.harnessList.Title,
		ModelTitle:          m.modelList.Title,
		AgentTitle:          m.agentList.Title,
	}

	if m.hoveredAgentID != "" {
		if agent, ok := m.agents[m.hoveredAgentID]; ok && agent != nil && agent.Info != nil {
			info := agent.Info
			cfg.TicketView = matrixTicketLaunchContextValue(info.TicketID, info.TicketTitle)
			cfg.HarnessView = matrixSingleValue(info.HarnessName)
			cfg.ModelView = matrixSingleValue(info.ModelName)
			cfg.AgentView = matrixSingleValue(info.AgentName)
			cfg.ModelColumnDisabled = false
			cfg.AgentColumnDisabled = false
		}
	}

	return cfg
}

func matrixSingleValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "(unknown)"
	}
	return trimmed
}

func matrixTicketLaunchContextValue(ticketID, ticketTitle string) string {
	id := strings.TrimSpace(ticketID)
	title := strings.TrimSpace(ticketTitle)

	switch {
	case id == "" && title == "":
		return "(unknown)"
	case id == "":
		return title
	case title == "":
		return id
	default:
		return id + ": " + title
	}
}

func (m UIModel) View() string {
	s := m.renderMainContent()

	footerStyle := lipgloss.NewStyle().
		Width(m.layout.Width).
		Background(ThemeFooterBg).
		Foreground(ThemeFooterFg).
		Padding(0, 1)

	helpView := m.help.View(m.keys)

	if m.refreshedRecently {
		hourglassFrames := []string{"󰊓", "󰊯", "󰊰", "󰊱"}
		var refreshIcon string
		if m.app.Fonts.HasNerdFont {
			refreshIcon = hourglassFrames[m.refreshAnimationFrame]
		} else {
			refreshIcon = "⟳"
		}
		helpView = refreshIcon + " Tickets refreshed  " + helpView
	}

	helpView = footerStyle.Render(helpView)

	theme := m.currentTheme
	if theme == nil {
		theme = &MatrixTheme
	}

	mainContentStyle := lipgloss.NewStyle().
		Width(m.layout.Width).
		Height(m.layout.Height).
		MaxHeight(m.layout.Height).
		Background(theme.AppBg).
		Foreground(theme.AppFg)

	mainContent := mainContentStyle.Render(s)

	// 1. Create a fully rectangular block of inner content
	fullView := lipgloss.JoinVertical(lipgloss.Top, mainContent, helpView)

	// Since docStyle natively uses standard spaces for padding, Bubble Tea will strip them
	// at the end of lines if it misidentifies the default terminal color. To avoid this,
	// we skip docStyle padding and instead use lipgloss.Place to pad the terminal directly
	// using non-breaking spaces (`\u00A0`).
	//
	// Because every line will end with `\u00A0` instead of a standard space, Bubble Tea's
	// trailing whitespace stripper will be bypassed, and the inner blocks' background
	// colored spaces will be fully preserved!
	if m.layout.TermWidth > 0 && m.layout.TermHeight > 0 {
		return lipgloss.Place(
			m.layout.TermWidth,
			m.layout.TermHeight,
			lipgloss.Center,
			lipgloss.Center,
			fullView,
			lipgloss.WithWhitespaceChars("\u00A0"),
			lipgloss.WithWhitespaceBackground(theme.AppBg),
			lipgloss.WithWhitespaceForeground(theme.AppFg),
		)
	}

	return fullView
}

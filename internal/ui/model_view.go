package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m *UIModel) updateSizes() {
	if m.width == 0 || m.height == 0 {
		return
	}

	listHeight := m.height - filterHeight
	innerListHeight := listHeight - 2

	var usableWidth int
	if m.showSidebar {
		usableWidth = m.width - 8
	} else {
		usableWidth = m.width - 6
	}

	baseX := usableWidth / 4

	if m.showSidebar {
		m.sidebarWidth = baseX
		m.tWidth = baseX
		m.hWidth = baseX / 2
		m.mWidth = baseX
		aWidth := usableWidth - (m.sidebarWidth + m.tWidth + m.hWidth + m.mWidth)
		if aWidth < 10 {
			aWidth = 10
		}
		m.aWidth = aWidth
	} else {
		m.sidebarWidth = 0
		m.tWidth = baseX
		m.hWidth = baseX
		m.mWidth = baseX
		m.aWidth = usableWidth - (m.tWidth + m.hWidth + m.mWidth)
	}

	safeW := func(w int) int {
		if w-2 < 1 {
			return 1
		}
		return w - 2
	}

	m.ticketList.SetSize(safeW(m.tWidth), innerListHeight)
	m.harnessList.SetSize(safeW(m.hWidth), innerListHeight)
	m.modelList.SetSize(safeW(m.mWidth), innerListHeight)
	m.agentList.SetSize(safeW(m.aWidth), innerListHeight)
	m.sidebar.SetSize(m.sidebarWidth, m.height)
	m.help.Width = m.width
}

func (m UIModel) renderMainContent() string {
	return RenderMainContent(MainContentConfig{
		State:               m.state,
		Focus:               m.focus,
		Loading:             m.loading,
		ShowFilePicker:      m.showFilePicker,
		ShowAddProjectModal: m.showAddProjectModal,
		ViewingAgentID:      m.viewingAgentID,
		Selection:           m.selection,
		Renderer:            m.app.Renderer,
		DryRun:              m.app.opts.DryRun,
		SelectedWorktree:    m.selectedWorktree,
		CurrentTheme:        m.currentTheme,
		ShowModal:           m.showModal,
		ModalContent:        m.modalContent,
		PendingProjectPath:  m.pendingProjectPath,
		Warnings:            m.warnings,
		Width:               m.width,
		Height:              m.height,
		Err:                 m.err,
		RetryStore:          m.retryStore,
		MatrixConfig:        m.buildMatrixConfig(),
		Agent:               m.agents[m.viewingAgentID],
		Filepicker:          m.filepicker,
		AnimState:           m.animState,
	})
}

func (m UIModel) buildMatrixConfig() MatrixConfig {
	return MatrixConfig{
		Width:               m.width,
		Height:              m.height,
		ShowSidebar:         m.showSidebar,
		SidebarWidth:        m.sidebarWidth,
		TWidth:              m.tWidth,
		HWidth:              m.hWidth,
		MWidth:              m.mWidth,
		AWidth:              m.aWidth,
		ModelColumnDisabled: m.modelColumnDisabled,
		AgentColumnDisabled: m.agentColumnDisabled,
		Focus:               m.focus,
		AnimState:           m.animState,
		Theme:               m.currentTheme,
		TicketView:          m.ticketList.View(),
		HarnessView:         m.harnessList.View(),
		ModelView:           m.modelList.View(),
		AgentView:           m.agentList.View(),
		SidebarView:         m.sidebar.View(),
		TicketTitle:         m.ticketList.Title,
		HarnessTitle:        m.harnessList.Title,
		ModelTitle:          m.modelList.Title,
		AgentTitle:          m.agentList.Title,
	}
}

func (m UIModel) View() string {
	s := m.renderMainContent()

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
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
		Width(m.width).
		Height(m.height).
		MaxHeight(m.height).
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
	if m.termWidth > 0 && m.termHeight > 0 {
		return lipgloss.Place(
			m.termWidth,
			m.termHeight,
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

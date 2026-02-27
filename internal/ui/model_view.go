package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbust/internal/domain"
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
	var s string
	switch m.state {
	case ViewStateMatrix:
		if m.loading {
			s = "Loading tickets...\n"
		} else if m.viewingAgentID != "" {
			// Show agent output view
			s = m.renderAgentOutputView()
		} else {
			// Show the matrix view
			s = m.renderMatrixView()
		}
	case ViewStateConfirm:
		s = confirmView(m.selection, m.app.Renderer, m.app.opts.DryRun, m.selectedWorktree)
	case ViewStateError:
		s = errorView(m.err)
	}

	if m.showModal {
		s = lipgloss.NewStyle().Faint(true).Render(s)

		modalWidth := m.width - 10
		if modalWidth < 40 {
			modalWidth = 40
		}

		modalBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ThemeActive).
			Padding(1, 2).
			Width(modalWidth).
			Render(m.modalContent)

		s = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalBox)
	}

	if len(m.warnings) > 0 {
		warningStyle := lipgloss.NewStyle().Foreground(ThemeWarning).MarginTop(1)
		for _, w := range m.warnings {
			s += "\n" + warningStyle.Render("⚠ "+w)
		}
	}
	return s
}

func (m UIModel) renderMatrixView() string {
	listHeight := m.height - filterHeight

	activeBorder := func(w int) lipgloss.Style {
		if w < 2 {
			w = 2
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ThemeActive).
			Width(w - 2).
			Height(listHeight - 2)
	}

	inactiveBorder := func(w int) lipgloss.Style {
		if w < 2 {
			w = 2
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ThemeInactive).
			Faint(false).
			Width(w - 2).
			Height(listHeight - 2)
	}

	var tView, hView, mView, aView string

	if m.focus == FocusTickets {
		tView = activeBorder(m.tWidth).Render(m.ticketList.View())
	} else {
		tView = inactiveBorder(m.tWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.ticketList.View()))
	}

	if m.focus == FocusHarness {
		hView = activeBorder(m.hWidth).Render(m.harnessList.View())
	} else {
		hView = inactiveBorder(m.hWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.harnessList.View()))
	}

	if m.focus == FocusModel {
		mView = activeBorder(m.mWidth).Render(m.modelList.View())
	} else {
		mView = inactiveBorder(m.mWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.modelList.View()))
	}

	if m.focus == FocusAgent {
		aView = activeBorder(m.aWidth).Render(m.agentList.View())
	} else {
		aView = inactiveBorder(m.aWidth).Render(lipgloss.NewStyle().Faint(true).Render(m.agentList.View()))
	}

	matrixWidth := m.tWidth + m.hWidth + m.mWidth + m.aWidth + 6

	filterBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(matrixWidth-2).
		Height(1).
		Padding(0, 1).
		Render("Filters: [All] | (Press / to search - Reactive Filter bb-0vw pending)")

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

	if m.showSidebar {
		w := m.sidebarWidth
		if w < 2 {
			w = 2
		}

		sidebarBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Width(w - 2).
			Height(m.height - 2)

		if m.focus == FocusSidebar {
			sidebarBorder = sidebarBorder.BorderForeground(ThemeActive)
		} else {
			sidebarBorder = sidebarBorder.BorderForeground(ThemeInactive)
		}

		sidebarContent := m.sidebar.View()
		sidebarBox := sidebarBorder.Render(sidebarContent)

		return lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, lipgloss.NewStyle().Width(2).Render("  "), rightPanelBox)
	}

	return rightPanelBox
}

func (m UIModel) renderAgentOutputView() string {
	agent, ok := m.agents[m.viewingAgentID]
	if !ok {
		return "Agent not found\n\n[Press back to return]"
	}

	var statusStr string
	var statusColor lipgloss.Color
	switch agent.Info.Status {
	case domain.AgentRunning:
		statusStr = "Running"
		statusColor = lipgloss.Color("34")
	case domain.AgentCompleted:
		statusStr = "Completed"
		statusColor = lipgloss.Color("245")
	case domain.AgentFailed:
		statusStr = "Failed"
		statusColor = lipgloss.Color("9")
	default:
		statusStr = "Unknown"
		statusColor = lipgloss.Color("245")
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)

	header := headerStyle.Render(fmt.Sprintf("Agent: %s", agent.Info.Name))
	statusLine := fmt.Sprintf("Status: %s", statusStyle.Render(statusStr))
	windowLine := fmt.Sprintf("Window: %s", agent.Info.WindowName)

	// Show output if we have it, otherwise show placeholder
	var outputContent string
	if agent.LastOutput != "" {
		outputContent = agent.LastOutput
	} else if agent.Info.Status == domain.AgentRunning {
		outputContent = "Waiting for output..."
	} else {
		outputContent = "No output available"
	}

	outputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ThemeInactive).
		Width(m.width-4).
		Height(m.height-10).
		Padding(0, 1)

	content := lipgloss.JoinVertical(lipgloss.Top,
		header,
		statusLine,
		windowLine,
		"",
		"Output:",
		outputStyle.Render(outputContent),
		"",
		"[Press Enter to return to matrix]",
	)

	return content
}

func (m UIModel) View() string {
	s := m.renderMainContent()

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(ThemeFooterBg).
		Foreground(ThemeFooterFg).
		Padding(0, 1)

	helpView := footerStyle.Render(m.help.View(m.keys))

	mainContentStyle := lipgloss.NewStyle().Height(m.height)
	mainContent := mainContentStyle.Render(s)

	return docStyle.Render(lipgloss.JoinVertical(lipgloss.Top, mainContent, helpView))
}

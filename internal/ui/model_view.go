package ui

import (
	"fmt"
	"time"

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
			s = m.renderLoadingView()
		} else if m.viewingAgentID != "" {
			// Show agent output view
			s = m.renderAgentOutputView()
		} else {
			// Show the matrix view
			s = m.renderMatrixView()
		}
	case ViewStateConfirm:
		s = confirmView(m.selection, m.app.Renderer, m.app.opts.DryRun, m.selectedWorktree, m.currentTheme)
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
			Border(lipgloss.ThickBorder()).
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

// renderLoadingView displays an arcade-style loading screen
func (m UIModel) renderLoadingView() string {
	// Animated spinner frames
	frames := []string{"◜", "◝", "◞", "◟"}
	frameIndex := int(time.Since(m.animState.StartTime).Seconds()*4) % 4
	frame := frames[frameIndex]

	// Use theme colors for loading
	spinnerColor := m.currentTheme.TitleColor
	arcadeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.currentTheme.ArcadeGold)

	s := "\n\n"
	s += lipgloss.NewStyle().Foreground(spinnerColor).Render(frame+" Initializing...") + "\n\n"
	s += arcadeStyle.Render("INSERT COIN TO START") + "\n"
	s += lipgloss.NewStyle().Faint(true).Render("(Loading tickets...)")
	return s
}

func (m UIModel) renderMatrixView() string {
	// Guard against uninitialized dimensions
	if m.height < filterHeight+2 {
		return "Initializing..."
	}

	listHeight := m.height - filterHeight

	// Get the current theme
	theme := m.currentTheme
	if theme == nil {
		theme = &MatrixTheme
	}

	// Determine the active border color - use flash color if lock-in is active for focused column
	var activeColor lipgloss.Color
	if m.animState.shouldShowFlash(m.focus) {
		// Flash takes priority during the bright phase of the animation
		activeColor = FlashColor
	} else {
		// Use cycling color for more dynamic effect
		activeColor = getCyclingColor(m.animState.PulsePhase, m.animState.ColorCycleIndex, theme)
	}

	// Get glow color based on current pulse phase
	glowColor := getGlowColor(m.animState.PulsePhase, theme)

	activeBorder := func(w int) lipgloss.Style {
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

	inactiveBorder := func(w int) lipgloss.Style {
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

	// Constrain list views to prevent overflow: lipgloss Height only pads,
	// MaxHeight truncates. MaxWidth prevents wrapping in narrow columns where
	// the bubbles list may render 1 char wider than SetSize.
	capView := func(view string, w int) string {
		return lipgloss.NewStyle().MaxHeight(listHeight - 2).MaxWidth(w - 2).Render(view)
	}
	faintCapView := func(view string, w int) string {
		return lipgloss.NewStyle().Faint(true).MaxHeight(listHeight - 2).MaxWidth(w - 2).Render(view)
	}

	var tView, hView, mView, aView string

	// Add visible focus indicator (▶) to the title of the focused column
	// This allows teatest to detect focus changes via text output
	const focusIndicator = "▶ "
	const noIndicator = "  "

	// Style for focused column titles - golden arcade style
	focusedTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.FocusIndicator)

	ticketTitle := m.ticketList.Title
	harnessTitle := m.harnessList.Title
	modelTitle := m.modelList.Title
	agentTitle := m.agentList.Title

	if m.focus == FocusTickets {
		m.ticketList.Title = focusedTitleStyle.Render(focusIndicator) + ticketTitle
		tView = activeBorder(m.tWidth).Render(capView(m.ticketList.View(), m.tWidth))
		m.ticketList.Title = ticketTitle // Restore original title
	} else {
		m.ticketList.Title = noIndicator + ticketTitle
		tView = inactiveBorder(m.tWidth).Render(faintCapView(m.ticketList.View(), m.tWidth))
		m.ticketList.Title = ticketTitle // Restore original title
	}

	if m.focus == FocusHarness {
		m.harnessList.Title = focusedTitleStyle.Render(focusIndicator) + harnessTitle
		hView = activeBorder(m.hWidth).Render(capView(m.harnessList.View(), m.hWidth))
		m.harnessList.Title = harnessTitle // Restore original title
	} else {
		m.harnessList.Title = noIndicator + harnessTitle
		hView = inactiveBorder(m.hWidth).Render(faintCapView(m.harnessList.View(), m.hWidth))
		m.harnessList.Title = harnessTitle // Restore original title
	}

	// Model column - greyed out if disabled
	if m.modelColumnDisabled {
		disabledStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(ThemeInactive).
			Faint(true).
			Width(m.mWidth-2).
			Height(listHeight-2).
			Align(lipgloss.Center, lipgloss.Center)
		mView = disabledStyle.Render("N/A\n\nNo models available\nfor this harness")
	} else if m.focus == FocusModel {
		mView = activeBorder(m.mWidth).Render(capView(m.modelList.View(), m.mWidth))
		m.modelList.Title = modelTitle // Restore original title
	} else {
		m.modelList.Title = noIndicator + modelTitle
		mView = inactiveBorder(m.mWidth).Render(faintCapView(m.modelList.View(), m.mWidth))
		m.modelList.Title = modelTitle // Restore original title
	}

	// Agent column - greyed out if disabled
	if m.agentColumnDisabled {
		disabledStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(ThemeInactive).
			Faint(true).
			Width(m.aWidth-2).
			Height(listHeight-2).
			Align(lipgloss.Center, lipgloss.Center)
		aView = disabledStyle.Render("N/A\n\nNo agents available\nfor this harness")
	} else if m.focus == FocusAgent {
		aView = activeBorder(m.aWidth).Render(capView(m.agentList.View(), m.aWidth))
		m.agentList.Title = agentTitle // Restore original title
	} else {
		m.agentList.Title = noIndicator + agentTitle
		aView = inactiveBorder(m.aWidth).Render(faintCapView(m.agentList.View(), m.aWidth))
		m.agentList.Title = agentTitle // Restore original title
	}

	matrixWidth := m.tWidth + m.hWidth + m.mWidth + m.aWidth + 6

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

	if m.showSidebar {
		w := m.sidebarWidth
		if w < 2 {
			w = 2
		}

		sidebarBorder := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			Width(w - 2).
			Height(m.height - 2)

		if m.focus == FocusSidebar {
			sidebarBorder = sidebarBorder.BorderForeground(activeColor)
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
		Border(lipgloss.ThickBorder()).
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

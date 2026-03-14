package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/data/dolt"
)

func (m UIModel) handleModalKeyMsg() (tea.Model, tea.Cmd, bool) {
	if m.showModal {
		m.showModal = false
		return m, nil, true
	}
	return m, nil, false
}

func (m UIModel) handleQuitKeyMsg() (tea.Model, tea.Cmd, bool) {
	if m.state == ViewStateAgentOutput {
		m.viewingAgentID = ""
		m.state = ViewStateMatrix
		return m, nil, true
	}
	return m, tea.Quit, true
}

func (m UIModel) handleRefreshKeyMsg() (tea.Model, tea.Cmd, bool) {
	if m.state == ViewStateMatrix && m.focus == FocusTickets {
		m.state = ViewStateLoading
		return m, tea.Batch(loadTicketsCmd(m.app.Project().Store()), discoverWorktreesCmd(m.app)), true
	}
	return m, nil, false
}

func (m UIModel) handleBackKeyMsg() (tea.Model, tea.Cmd, bool) {
	if m.state == ViewStateConfirm {
		m.state = ViewStateMatrix
		return m, nil, true
	}
	if m.state == ViewStateAgentOutput {
		m.viewingAgentID = ""
		m.state = ViewStateMatrix
		return m, nil, true
	}
	if m.state == ViewStateMatrix && m.focus > FocusTickets {
		m.focus--
		return m, nil, true
	}
	return m, nil, false
}

func (m UIModel) handleInfoKeyMsg() (tea.Model, tea.Cmd, bool) {
	if m.state == ViewStateMatrix && m.focus == FocusTickets {
		if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
			m.showModal = true
			m.modalContent = "Loading bd show..."
			return m, loadModalCmd(i.ticket.ID), true
		}
	}
	return m, nil, false
}

func (m UIModel) handleToggleSidebarKeyMsg() (tea.Model, tea.Cmd, bool) {
	m.showSidebar = !m.showSidebar
	m.updateSizes()
	return m, nil, true
}

func (m UIModel) handleToggleThemeKeyMsg() (tea.Model, tea.Cmd, bool) {
	m.animState.nextTheme()
	m.currentTheme = m.animState.getCurrentTheme()
	// Ticket list uses the dynamic ticketDelegate; update its theme in-place
	// so width state is preserved.
	if m.ticketDel != nil {
		m.ticketDel.applyTheme(m.currentTheme)
	} else {
		m.ticketList.SetDelegate(newGradientDelegate(m.currentTheme))
	}
	m.harnessList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.modelList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.agentList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.dirtyTicket = true
	m.dirtyHarness = true
	m.dirtyModel = true
	m.dirtyAgent = true
	return m, nil, true
}

func (m UIModel) handleFilePickerKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != ViewStateFilePicker {
		return m, nil, false
	}
	switch msg.String() {
	case "a":
		currentDir := m.filepicker.CurrentDirectory
		if currentDir != "" {
			return m, m.checkAndPromptAddProject(currentDir), true
		}
		return m, nil, true
	case "esc":
		m.state = ViewStateMatrix
		return m, nil, true
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	return m, cmd, true
}

func (m UIModel) handleAddProjectModalKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != ViewStateAddProjectModal {
		return m, nil, false
	}
	switch msg.String() {
	case "y", "Y":
		return m, func() tea.Msg {
			return addProjectConfirmedMsg{path: m.pendingProjectPath}
		}, true
	case "n", "N", "q", "esc":
		return m, func() tea.Msg {
			return addProjectCancelledMsg{}
		}, true
	}
	return m, nil, true
}

func (m UIModel) handleErrorStateKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != ViewStateError {
		return m, nil, false
	}
	switch msg.String() {
	case "q", "Q":
		return m, tea.Quit, true
	case "r", "R":
		if m.retryStore != nil {
			m.state = ViewStateLoading
			return m, loadTicketsCmd(m.retryStore), true
		}
	case "s", "S":
		if m.retryStore != nil {
			if doltStore, ok := m.retryStore.(*dolt.Store); ok {
				if doltStore.CanRetryConnection() {
					m.state = ViewStateLoading
					return m, startServerAndRetryCmd(m.app, doltStore), true
				}
			}
		}
	}
	return m, nil, true
}

func (m UIModel) handleGlobalKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if key.Matches(msg, m.keys.Quit) {
		return m.handleQuitKeyMsg()
	}

	if key.Matches(msg, m.keys.Refresh) {
		if model, cmd, handled := m.handleRefreshKeyMsg(); handled {
			return model, cmd, true
		}
	}

	if key.Matches(msg, m.keys.Back) {
		if model, cmd, handled := m.handleBackKeyMsg(); handled {
			return model, cmd, true
		}
	}

	if key.Matches(msg, m.keys.Info) {
		if model, cmd, handled := m.handleInfoKeyMsg(); handled {
			return model, cmd, true
		}
	}

	if key.Matches(msg, m.keys.ToggleSidebar) {
		return m.handleToggleSidebarKeyMsg()
	}

	if key.Matches(msg, m.keys.ToggleTheme) {
		return m.handleToggleThemeKeyMsg()
	}

	return m, nil, false
}

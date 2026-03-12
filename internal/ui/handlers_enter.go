package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/domain"
)

// handleEnterKey processes Enter key based on current view state
func (m UIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	// Exit agent output view when Enter is pressed
	if m.state == ViewStateAgentOutput {
		m.viewingAgentID = ""
		m.state = ViewStateMatrix
		return m, nil
	}

	switch m.state {
	case ViewStateMatrix:
		return m.handleMatrixEnterKey()
	case ViewStateConfirm:
		m.state = ViewStateMatrix
		return m, m.launchCmd()
	}
	return m, nil
}

// handleMatrixEnterKey routes Enter key to the appropriate handler based on focus
func (m UIModel) handleMatrixEnterKey() (tea.Model, tea.Cmd) {
	switch m.focus {
	case FocusSidebar:
		return m.handleSidebarEnterKey()
	case FocusTickets:
		return m.handleTicketsEnterKey()
	case FocusHarness:
		return m.handleHarnessEnterKey()
	case FocusModel:
		return m.handleModelEnterKey()
	case FocusAgent:
		return m.handleAgentEnterKey()
	}
	return m, nil
}

// handleSidebarEnterKey handles Enter key when sidebar is focused
func (m UIModel) handleSidebarEnterKey() (tea.Model, tea.Cmd) {
	node := m.sidebar.State().CurrentNode()
	if node != nil && node.Type == domain.NodeTypeWorktree {
		m.selectedWorktree = node.Path
		m.sidebar.SetSelectedPath(node.Path)
		m.focus = FocusTickets
		m.sidebar.SetFocused(false)
		return m, nil
	}
	if node != nil && len(node.Children) > 0 {
		m.sidebar.State().ToggleExpand()
	}
	return m, nil
}

// handleTicketsEnterKey handles Enter key when tickets column is focused
func (m UIModel) handleTicketsEnterKey() (tea.Model, tea.Cmd) {
	if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
		m.selection.Ticket = i.ticket

		if len(m.harnesses) == 1 {
			m.selection.Harness = m.harnesses[0]
			m, _ = m.handleModelSkip()
		}

		if m.focus < FocusAgent {
			m.advanceFocus()
		}
		return m, nil
	}
	return m, nil
}

// handleHarnessEnterKey handles Enter key when harness column is focused
func (m UIModel) handleHarnessEnterKey() (tea.Model, tea.Cmd) {
	if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
		m.selection.Harness = i.harness
		m, _ = m.handleModelSkip()
		m, _ = m.handleAgentSkip()
		if m.focus < FocusAgent {
			m.advanceFocus()
		}
		return m, nil
	}
	return m, nil
}

// handleModelEnterKey handles Enter key when model column is focused
func (m UIModel) handleModelEnterKey() (tea.Model, tea.Cmd) {
	if i, ok := m.modelList.SelectedItem().(modelItem); ok {
		m.selection.Model = i.name
		m, _ = m.handleAgentSkip()
		if m.focus < FocusAgent {
			m.advanceFocus()
		}
		return m, nil
	}
	return m, nil
}

// handleAgentEnterKey handles Enter key when agent column is focused
func (m UIModel) handleAgentEnterKey() (tea.Model, tea.Cmd) {
	if i, ok := m.agentList.SelectedItem().(agentItem); ok {
		m.selection.Agent = i.name
		m.state = ViewStateConfirm
		return m, nil
	}
	return m, nil
}

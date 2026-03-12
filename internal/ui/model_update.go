package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m UIModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if model, cmd, handled := m.handleFilePickerKeyMsg(msg); handled {
		return model, cmd, handled
	}

	if model, cmd, handled := m.handleAddProjectModalKeyMsg(msg); handled {
		return model, cmd, handled
	}

	if model, cmd, handled := m.handleErrorStateKeyMsg(msg); handled {
		return model, cmd, handled
	}

	if model, cmd, handled := m.handleModalKeyMsg(); handled {
		return model, cmd, true
	}

	if model, cmd, handled := m.handleGlobalKeyMsg(msg); handled {
		return model, cmd, true
	}

	if model, cmd, handled := m.handleNavigationKeysMsg(msg); handled {
		return model, cmd, true
	}

	if key.Matches(msg, m.keys.Enter) {
		if m.focus == FocusSidebar {
			return m, nil, false
		}

		flashCmd := lockInCmd(m.focus)

		model, cmd := m.handleEnterKey()
		return model, tea.Batch(flashCmd, cmd), true
	}

	if model, cmd, handled := m.HandleSidebarAgentKeysMsg(msg); handled {
		return model, cmd, true
	}

	return m, nil, false
}

func (m *UIModel) updateKeyBindings() {
	switch m.state {
	case ViewStateMatrix:
		switch m.focus {
		case FocusSidebar:
			m.keys.Back.SetEnabled(false)
			m.keys.Refresh.SetEnabled(false)
			m.keys.Info.SetEnabled(false)
			m.keys.Enter.SetEnabled(true)
		case FocusTickets:
			m.keys.Back.SetEnabled(false)
			m.keys.Refresh.SetEnabled(true)
			m.keys.Info.SetEnabled(true)
			m.keys.Enter.SetEnabled(true)
		default:
			m.keys.Back.SetEnabled(true)
			m.keys.Refresh.SetEnabled(false)
			m.keys.Info.SetEnabled(false)
			m.keys.Enter.SetEnabled(true)
		}
		m.keys.ToggleSidebar.SetEnabled(true)
		m.keys.ToggleTheme.SetEnabled(true)
	case ViewStateError:
		m.keys.Back.SetEnabled(false)
		m.keys.Refresh.SetEnabled(false)
		m.keys.Enter.SetEnabled(false)
		m.keys.Info.SetEnabled(false)
		m.keys.ToggleSidebar.SetEnabled(false)
		m.keys.ToggleTheme.SetEnabled(false)
	default:
		m.keys.Back.SetEnabled(true)
		m.keys.Refresh.SetEnabled(false)
		m.keys.Enter.SetEnabled(true)
		m.keys.Info.SetEnabled(false)
		m.keys.ToggleSidebar.SetEnabled(false)
		m.keys.ToggleTheme.SetEnabled(true)
	}
}

func updateListCaches(m *UIModel) UIModel {
	if m.dirtyTicket || !m.initializedTicket {
		m.ticketViewCache = m.ticketList.View()
		m.dirtyTicket = false
		m.initializedTicket = true
	}
	if m.dirtyHarness || !m.initializedHarness {
		m.harnessViewCache = m.harnessList.View()
		m.dirtyHarness = false
		m.initializedHarness = true
	}
	if m.dirtyModel || !m.initializedModel {
		m.modelViewCache = m.modelList.View()
		m.dirtyModel = false
		m.initializedModel = true
	}
	if m.dirtyAgent || !m.initializedAgent {
		m.agentViewCache = m.agentList.View()
		m.dirtyAgent = false
		m.initializedAgent = true
	}
	return *m
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Advance sidebar animation per event to ensure glitch effect runs
	// at a rate proportional to overall UI activity, matching old behavior.
	m.sidebar.TickAnimation()

	if m.state == ViewStateFilePicker {
		switch msg.(type) {
		case tea.KeyMsg, tea.WindowSizeMsg:
			// Let normal flow handle it so we process app-level keys and resize
		default:
			var fpCmd tea.Cmd
			m.filepicker, fpCmd = m.filepicker.Update(msg)
			if fpCmd != nil {
				return m, fpCmd
			}
		}
	}

	if newModel, cmd, handled := m.handleCoreMsgs(msg); handled {
		if uiModel, ok := newModel.(UIModel); ok {
			newModel = updateListCaches(&uiModel)
		}
		return newModel, cmd
	}
	if newModel, cmd, handled := m.handleProjectMsgs(msg); handled {
		if uiModel, ok := newModel.(UIModel); ok {
			newModel = updateListCaches(&uiModel)
		}
		return newModel, cmd
	}
	if newModel, cmd, handled := m.handleAgentMsgs(msg); handled {
		if uiModel, ok := newModel.(UIModel); ok {
			newModel = updateListCaches(&uiModel)
		}
		return newModel, cmd
	}

	uiModel, cmd := m.handleFocusUpdate(msg)
	uiModel.updateKeyBindings()
	newModel := updateListCaches(&uiModel)
	return newModel, cmd
}

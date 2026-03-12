package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Handler Registration and Message Flow
//
// This file contains the core Update() orchestration logic for the UI.
// The Update() function is the main message dispatcher that routes incoming
// messages to appropriate handlers.
//
// Message Flow:
//
// 1. Animation Tick (always): Advancing sidebar animation for visual effects
// 2. File Picker State: Special handling for file picker mode
// 3. Core Messages: handleCoreMsgs() handles:
//    - registryLoadedMsg: Initial registry load
//    - ticketsLoadedMsg: Ticket data loaded
//    - errMsg/warningMsg: Error/warning display
//    - modalContentMsg: Modal content updates
//    - tea.WindowSizeMsg: Window resize events
//    - tea.KeyMsg: Keyboard input (dispatched via handleKeyMsg)
//
// 4. Project Messages: handleProjectMsgs() handles:
//    - worktreesDiscoveredMsg: Worktree discovery results
//    - runningAgentsLoadedMsg: Running agents loaded
//    - WorktreeSelectedMsg: Worktree selection change
//    - serverStartedMsg: Server started notification
//    - OpenFilePickerMsg: Open file picker
//    - ShowAddProjectModalMsg: Show add project modal
//    - addProjectConfirmedMsg/CancelledMsg: Add project actions
//    - filepicker.RecentsChangedMsg: Recent directories changed
//
// 5. Agent Messages: handleAgentMsgs() handles:
//    - launchResultMsg: Agent launch result
//    - AgentHoveredMsg/AgentHoverEndedMsg: Agent hover state
//    - AgentSelectedMsg: Agent selection
//    - AgentStatusMsg: Agent status updates
//    - agentTickMsg: Agent periodic updates
//    - agentOutputMsg: Agent output
//    - animationTickMsg: Animation ticks
//    - lockInMsg: Column lock-in animation
//    - AgentClearedMsg/AllStoppedAgentsClearedMsg: Agent clearing
//    - ticketUpdateCheckMsg/ticketUpdateCheckNeededMsg: Ticket updates
//    - ticketsAutoRefreshedMsg/clearRefreshIndicatorMsg/refreshAnimationTickMsg: Refresh handling
//
// 6. Focus Update: handleFocusUpdate() handles focus-specific updates based on current focus
//    - FocusSidebar: Sidebar cursor and selection
//    - FocusTickets: Ticket list cursor
//    - FocusHarness: Harness list cursor and selection
//    - FocusModel: Model list cursor
//    - FocusAgent: Agent list cursor
//
// Key Handler Dispatch:
//
// Key messages are dispatched through handleKeyMsg() in priority order:
// 1. File picker keys (handleFilePickerKeyMsg)
// 2. Add project modal keys (handleAddProjectModalKeyMsg)
// 3. Error state keys (handleErrorStateKeyMsg)
// 4. Modal keys (handleModalKeyMsg)
// 5. Global keys (handleGlobalKeyMsg)
// 6. Navigation keys (handleNavigationKeysMsg)
// 7. Enter key (special handling with lock-in animation)
// 8. Sidebar agent keys (HandleSidebarAgentKeysMsg)
//
// Caching Strategy:
//
// List views are cached for performance:
// - ticketViewCache: Ticket list view (dirtyTicket flag)
// - harnessViewCache: Harness list view (dirtyHarness flag)
// - modelViewCache: Model list view (dirtyModel flag)
// - agentViewCache: Agent list view (dirtyAgent flag)
//
// updateListCaches() is called after each handler returns to update dirty caches.
// Dirty flags are set whenever a list's state changes (cursor, selection, data).
//
// Key Bindings:
//
// updateKeyBindings() enables/disables key bindings based on current state and focus:
// - ViewStateMatrix: Matrix view state with focus-specific bindings
// - ViewStateError: Error state (most keys disabled)
// - Other states: Minimal key bindings enabled

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

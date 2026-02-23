package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/discovery"
)

func (m UIModel) handleTicketsLoaded(msg ticketsLoadedMsg) (tea.Model, tea.Cmd) {
	if len(msg) == 0 {
		m.ticketList = newEmptyTicketList()
	} else {
		m.ticketList = newTicketList(msg)
	}
	initList(&m.ticketList, m.width, m.height, "Select a Ticket")
	m.loading = false
	return m, nil
}

func (m UIModel) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	m.loading = false
	m.state = ViewStateError
	return m, nil
}

func (m UIModel) handleWarningMsg(msg warningMsg) (tea.Model, tea.Cmd) {
	m.warnings = append(m.warnings, msg.err.Error())
	return m, nil
}

func (m UIModel) handleLaunchResult(msg launchResultMsg) (tea.Model, tea.Cmd) {
	m.launchResult = msg.res
	m.err = msg.err
	m.state = ViewStateResult

	if msg.err == nil && msg.res != nil && msg.res.WindowName != "" {
		m.monitoringWindow = msg.res.WindowName
		return m, tea.Batch(
			m.pollWindowStatusCmd(msg.res.WindowName),
			m.startMonitoringCmd(msg.res.WindowName),
		)
	}
	return m, nil
}

func (m UIModel) handleStatusUpdate(msg statusUpdateMsg) (tea.Model, tea.Cmd) {
	m.windowStatus = msg.status
	m.windowStatusEmoji = msg.emoji
	return m, nil
}

func (m UIModel) handleTickMsg(msg tickMsg) (tea.Model, tea.Cmd) {
	if m.state == ViewStateResult && m.monitoringWindow == msg.windowName {
		return m, tea.Batch(
			m.pollWindowStatusCmd(msg.windowName),
			m.startMonitoringCmd(msg.windowName),
		)
	}
	return m, nil
}

func (m UIModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (UIModel, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	m.width, m.height = msg.Width-h, msg.Height-v-footerHeight

	if m.width < minWindowWidth {
		m.width = minWindowWidth
	}
	if m.height < minWindowHeight {
		m.height = minWindowHeight
	}

	m.updateSizes()
	return m, nil
}

func (m UIModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.showModal {
		if key.Matches(msg, m.keys.Back, m.keys.Quit, m.keys.Enter, m.keys.Info) {
			m.showModal = false
		}
		return m, nil, true
	}

	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit, true
	}
	if key.Matches(msg, m.keys.Refresh) {
		if m.state == ViewStateMatrix && m.focus == FocusTickets {
			m.loading = true
			return m, loadTicketsCmd(m.app.store), true
		}
	}
	if key.Matches(msg, m.keys.Back) {
		if m.state == ViewStateConfirm {
			m.state = ViewStateMatrix
			return m, nil, true
		}
		if m.state == ViewStateMatrix && m.focus > FocusTickets {
			m.focus--
			return m, nil, true
		}
	}
	if key.Matches(msg, m.keys.Info) {
		if m.state == ViewStateMatrix && m.focus == FocusTickets {
			if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
				m.showModal = true
				m.modalContent = "Loading bd show..."
				return m, loadModalCmd(i.ticket.ID), true
			}
		}
	}
	if key.Matches(msg, m.keys.ToggleSidebar) {
		m.showSidebar = !m.showSidebar
		m.updateSizes()
		return m, nil, true
	}

	switch msg.String() {
	case "left":
		if m.state == ViewStateMatrix && m.focus > FocusTickets {
			m.focus--
			return m, nil, true
		}
	case "right":
		if m.state == ViewStateMatrix && m.focus < FocusAgent {
			m.focus++
			return m, nil, true
		}
	}

	if key.Matches(msg, m.keys.Enter) {
		model, cmd := m.handleEnterKey()
		return model, cmd, true
	}

	return m, nil, false
}

func (m UIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.state {
	case ViewStateMatrix:
		switch m.focus {
		case FocusTickets:
			if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
				m.selection.Ticket = i.ticket

				if len(m.harnesses) == 1 {
					m.selection.Harness = m.harnesses[0]
					m, _ = m.handleModelSkip()
				}

				if m.focus < FocusAgent {
					m.focus++
				}
				return m, nil
			}
		case FocusHarness:
			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				m.selection.Harness = i.harness
				m, _ = m.handleModelSkip()
				if m.focus < FocusAgent {
					m.focus++
				}
				return m, nil
			}
		case FocusModel:
			if i, ok := m.modelList.SelectedItem().(modelItem); ok {
				m.selection.Model = i.name
				m, _ = m.handleAgentSkip()
				if m.focus < FocusAgent {
					m.focus++
				}
				return m, nil
			}
		case FocusAgent:
			if i, ok := m.agentList.SelectedItem().(agentItem); ok {
				m.selection.Agent = i.name
				m.state = ViewStateConfirm
				return m, nil
			}
		}
	case ViewStateConfirm:
		m.state = ViewStateResult
		return m, m.launchCmd()
	case ViewStateResult:
		return m, tea.Quit
	}
	return m, nil
}

func (m UIModel) handleModelSkip() (UIModel, tea.Cmd) {
	models := m.selection.Harness.SupportedModels

	expandedModels := make([]string, 0, len(models))
	for _, model := range models {
		switch {
		case strings.HasPrefix(model, discovery.PrefixProvider):
			providerID := strings.TrimPrefix(model, discovery.PrefixProvider)
			providerModels := m.app.Registry.GetModelsForProvider(providerID)
			expandedModels = append(expandedModels, providerModels...)
		case model == discovery.KeywordDiscoverActive:
			activeModels := m.app.Registry.GetActiveModels()
			expandedModels = append(expandedModels, activeModels...)
		default:
			expandedModels = append(expandedModels, model)
		}
	}

	uniqueModels := make([]string, 0, len(expandedModels))
	seen := make(map[string]bool)
	for _, model := range expandedModels {
		if !seen[model] {
			seen[model] = true
			uniqueModels = append(uniqueModels, model)
		}
	}
	models = uniqueModels

	if len(models) == 0 {
		m.selection.Model = ""
	}
	m.modelList = newModelList(models)
	m.updateSizes()
	return m, nil
}

func (m UIModel) handleAgentSkip() (UIModel, tea.Cmd) {
	agents := m.selection.Harness.SupportedAgents
	if len(agents) == 0 {
		m.selection.Agent = ""
	}

	m.agentList = newAgentList(agents)
	m.updateSizes()
	return m, nil
}

func (m *UIModel) updateKeyBindings() {
	switch m.state {
	case ViewStateMatrix:
		if m.focus == FocusTickets {
			m.keys.Back.SetEnabled(false)
			m.keys.Refresh.SetEnabled(true)
			m.keys.Info.SetEnabled(true)
		} else {
			m.keys.Back.SetEnabled(true)
			m.keys.Refresh.SetEnabled(false)
			m.keys.Info.SetEnabled(false)
		}
		m.keys.ToggleSidebar.SetEnabled(true)
		m.keys.Enter.SetEnabled(true)
	case ViewStateResult, ViewStateError:
		m.keys.Back.SetEnabled(false)
		m.keys.Refresh.SetEnabled(false)
		m.keys.Enter.SetEnabled(false)
		m.keys.Info.SetEnabled(false)
		m.keys.ToggleSidebar.SetEnabled(false)
	default:
		m.keys.Back.SetEnabled(true)
		m.keys.Refresh.SetEnabled(false)
		m.keys.Enter.SetEnabled(true)
		m.keys.Info.SetEnabled(false)
		m.keys.ToggleSidebar.SetEnabled(false)
	}
}

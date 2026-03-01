package ui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

func (m UIModel) handleTicketsLoaded(msg ticketsLoadedMsg) (tea.Model, tea.Cmd) {
	if len(msg) == 0 {
		if m.app.Store() == nil {
			m.ticketList = createErrorList("Couldn't load ticket list:\nStore initialization failed")
			m.sidebar.SetStoreError(true)
			m.loading = false
			return m, nil
		}
		m.ticketList = newEmptyTicketList()
		m.sidebar.SetStoreError(false)
	} else {
		m.ticketList = newTicketList(msg)
		m.sidebar.SetStoreError(false)
	}
	initList(&m.ticketList, 0, 0, "Select a Ticket")
	m.loading = false
	m.updateSizes()
	m.lastTicketUpdate = time.Now()
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

	if msg.err != nil {
		m.state = ViewStateError
		return m, nil
	}

	if msg.res != nil && msg.res.WindowName != "" {
		// Create agent info
		agentID := msg.res.WindowName
		agentInfo := &domain.AgentInfo{
			ID:           agentID,
			Name:         m.selection.Ticket.ID,
			WindowName:   msg.res.WindowName,
			WindowID:     msg.res.WindowID,
			WorktreePath: m.selectedWorktree,
			Status:       domain.AgentRunning,
			StartedAt:    time.Now(),
			TicketID:     m.selection.Ticket.ID,
		}

		// Start output capture
		var capture *tmux.OutputCapture
		windowID := msg.res.WindowID
		if windowID == "" {
			// Fallback to window name if ID is empty
			windowID = msg.res.WindowName
		}
		if windowID != "" && m.app.Runner() != nil {
			capture = tmux.NewOutputCapture(m.app.Runner(), windowID)
			path, captureErr := capture.Start(context.Background())
			if captureErr != nil {
				// Log error but don't fail - agent still launched
				m.warnings = append(m.warnings, fmt.Sprintf("Failed to capture output: %v", captureErr))
				capture = nil
			}
			_ = path
		}

		// Register agent
		m.agents[agentID] = &RunningAgent{
			Info:    agentInfo,
			Capture: capture,
		}

		// Add agent node to sidebar under the worktree
		addAgentNodeToSidebar(&m, agentInfo)

		// Return to matrix instead of result screen
		m.state = ViewStateMatrix

		// Start monitoring the agent
		return m, tea.Batch(
			pollAgentStatusCmd(m.app, agentID, msg.res.WindowName),
			startAgentMonitoringCmd(agentID),
		)
	}

	// Return to matrix even on error (user can see error in sidebar status)
	m.state = ViewStateMatrix
	return m, nil
}

func (m UIModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (UIModel, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	// Account for: frame borders (v) + margins (verticalMargins) + footer
	m.width, m.height = msg.Width-h, msg.Height-v-verticalMargins-footerHeight

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
		if m.viewingAgentID != "" {
			m.viewingAgentID = ""
			return m, nil, true
		}
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
		// Exit agent output view
		if m.viewingAgentID != "" {
			m.viewingAgentID = ""
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

	// Handle manual left/right navigation outside of keys.go since it's intrinsic to the matrix
	switch msg.String() {
	case "left":
		if m.state == ViewStateMatrix && m.focus > FocusSidebar {
			m.retreatFocus()
			return m, nil, true
		}
	case "right":
		if m.state == ViewStateMatrix && m.focus < FocusAgent {
			m.advanceFocus()
			return m, nil, true
		}
	case "tab":
		if m.state == ViewStateMatrix {
			if m.focus < FocusAgent {
				m.advanceFocus()
			} else {
				m.focus = FocusSidebar
				m.sidebar.SetFocused(true)
			}
			return m, nil, true
		}
	}

	if key.Matches(msg, m.keys.Enter) {
		// Don't handle Enter if sidebar has focus - let sidebar handle it
		if m.focus == FocusSidebar {
			return m, nil, false
		}

		// Trigger lock-in flash before handling the selection
		// This provides immediate visual feedback that the button press was registered
		flashCmd := lockInCmd(m.focus)

		model, cmd := m.handleEnterKey()
		return model, tea.Batch(flashCmd, cmd), true
	}

	// Handle agent clearing keys
	if m.focus == FocusSidebar {
		switch msg.String() {
		case "c":
			// Clear selected agent (confirm if running)
			node := m.sidebar.State().CurrentNode()
			if node != nil && node.Type == domain.NodeTypeAgent && node.AgentInfo != nil {
				var capture *tmux.OutputCapture
				if agent, ok := m.agents[node.AgentInfo.ID]; ok {
					capture = agent.Capture
				}
				return m, clearAgentCmd(node.AgentInfo.ID, capture), true
			}
		case "C":
			// Clear all stopped agents
			var toClear []agentToClear
			for id, agent := range m.agents {
				if agent.Info.Status != domain.AgentRunning {
					toClear = append(toClear, agentToClear{id: id, capture: agent.Capture})
				}
			}
			if len(toClear) > 0 {
				return m, clearAllStoppedAgentsCmd(toClear), true
			}
			return m, nil, true
		}
	}

	return m, nil, false
}

func (m UIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	// Exit agent output view when Enter is pressed
	if m.viewingAgentID != "" {
		m.viewingAgentID = ""
		return m, nil
	}

	switch m.state {
	case ViewStateMatrix:
		switch m.focus {
		case FocusSidebar:
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
		case FocusTickets:
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
		case FocusHarness:
			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				// State transition: harness selection changed
				// When user selects a different harness, we need to re-evaluate which columns
				// should be disabled based on the new harness's SupportedModels and SupportedAgents.
				// This is done by calling handleModelSkip() and handleAgentSkip() which:
				// 1. Expand provider: and discover:active keywords to actual model/agent lists
				// 2. Set modelColumnDisabled/agentColumnDisabled flags based on list emptiness
				// 3. Clear any previously selected model/agent values when columns become disabled
				// 4. Update the UI lists to reflect the new available options
				//
				// Navigation automatically adapts: advanceFocus() and retreatFocus() check the
				// disabled flags and skip columns that have no selectable items.
				m.selection.Harness = i.harness
				m, _ = m.handleModelSkip()
				m, _ = m.handleAgentSkip()
				if m.focus < FocusAgent {
					m.advanceFocus()
				}
				return m, nil
			}
		case FocusModel:
			if i, ok := m.modelList.SelectedItem().(modelItem); ok {
				m.selection.Model = i.name
				m, _ = m.handleAgentSkip()
				if m.focus < FocusAgent {
					m.advanceFocus()
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
		m.state = ViewStateMatrix
		return m, m.launchCmd()
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

	// Set disabled flag based on whether there are any models
	m.modelColumnDisabled = len(models) == 0
	if m.modelColumnDisabled {
		m.selection.Model = ""
	}
	m.modelList = newModelList(models)
	m.updateSizes()
	return m, nil
}

func (m UIModel) handleAgentSkip() (UIModel, tea.Cmd) {
	agents := m.selection.Harness.SupportedAgents
	// Set disabled flag based on whether there are any agents
	m.agentColumnDisabled = len(agents) == 0
	if m.agentColumnDisabled {
		m.selection.Agent = ""
	}

	m.agentList = newAgentList(agents)
	m.updateSizes()
	return m, nil
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
	case ViewStateError:
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

func (m UIModel) handleWorktreesDiscovered(msg worktreesDiscoveredMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Worktree discovery: %v", msg.err))
		return m, nil
	}

	m.sidebar, _ = m.sidebar.Update(SidebarNodesMsg{Nodes: msg.nodes})

	if len(msg.nodes) > 0 && len(msg.nodes[0].Children) > 0 {
		initialPath := msg.nodes[0].Children[0].Path
		m.selectedWorktree = initialPath
		m.sidebar.SetSelectedPath(initialPath)
	}

	return m, nil
}

func (m UIModel) handleWorktreeSelected(msg WorktreeSelectedMsg) (tea.Model, tea.Cmd) {
	m.selectedWorktree = msg.Path
	m.sidebar.SetSelectedPath(msg.Path)
	m.focus = FocusTickets
	m.sidebar.SetFocused(false)
	return m, nil
}

// Agent management helpers

func addAgentNodeToSidebar(m *UIModel, agentInfo *domain.AgentInfo) {
	state := m.sidebar.State()
	for i := range state.Nodes {
		addAgentToProject(&state.Nodes[i], agentInfo)
	}
	state.RebuildFlatNodes()
}

func addAgentToProject(projectNode *domain.SidebarNode, agentInfo *domain.AgentInfo) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree && worktreeNode.Path == agentInfo.WorktreePath {
			agentNode := domain.SidebarNode{
				ID:         "agent-" + agentInfo.ID,
				Name:       agentInfo.Name,
				Path:       "agent:" + agentInfo.ID,
				Type:       domain.NodeTypeAgent,
				IsExpanded: false,
				IsRunning:  true,
				AgentInfo:  agentInfo,
				Children:   make([]domain.SidebarNode, 0),
			}
			worktreeNode.Children = append(worktreeNode.Children, agentNode)
			worktreeNode.IsExpanded = true
			return
		}
	}
}

func updateAgentNodeStatus(m *UIModel, agentID string, status domain.AgentStatus) {
	state := m.sidebar.State()
	for i := range state.FlatNodes {
		node := state.FlatNodes[i].Node
		if node.Type == domain.NodeTypeAgent && node.AgentInfo != nil && node.AgentInfo.ID == agentID {
			node.AgentInfo.Status = status
			node.IsRunning = status == domain.AgentRunning
			return
		}
	}
}

func (m UIModel) handleAgentSelected(msg AgentSelectedMsg) (tea.Model, tea.Cmd) {
	m.viewingAgentID = msg.AgentID

	var readOutputCmd tea.Cmd
	if agent, ok := m.agents[msg.AgentID]; ok {
		readOutputCmd = readAgentOutputCmd(msg.AgentID, agent.Capture)
	}

	return m, readOutputCmd
}

func (m UIModel) handleAgentStatus(msg AgentStatusMsg) (tea.Model, tea.Cmd) {
	if agent, ok := m.agents[msg.AgentID]; ok {
		agent.Info.Status = msg.Status
		updateAgentNodeStatus(&m, msg.AgentID, msg.Status)
	}
	return m, nil
}

func (m UIModel) handleAgentTick(msg agentTickMsg) (tea.Model, tea.Cmd) {
	agentID := msg.agentID
	agent, ok := m.agents[agentID]
	if !ok {
		return m, nil
	}

	// If viewing this agent, read output
	var readOutputCmd tea.Cmd
	if m.viewingAgentID == agentID {
		readOutputCmd = readAgentOutputCmd(agentID, agent.Capture)
	}

	// Continue monitoring if still running
	if agent.Info.Status == domain.AgentRunning {
		return m, tea.Batch(
			pollAgentStatusCmd(m.app, agentID, agent.Info.WindowName),
			startAgentMonitoringCmd(agentID),
			readOutputCmd,
		)
	}

	return m, readOutputCmd
}

func (m UIModel) handleAgentOutput(msg agentOutputMsg) (tea.Model, tea.Cmd) {
	if agent, ok := m.agents[msg.agentID]; ok {
		clean := ansi.Strip(msg.content)
		clean = strings.ReplaceAll(clean, "\r\n", "\n")
		clean = strings.ReplaceAll(clean, "\r", "\n")
		agent.LastOutput = clean
	}
	return m, nil
}

func (m UIModel) handleAgentCleared(msg AgentClearedMsg) (tea.Model, tea.Cmd) {
	// Remove from agents map
	delete(m.agents, msg.AgentID)

	// If we were viewing this agent, stop viewing
	if m.viewingAgentID == msg.AgentID {
		m.viewingAgentID = ""
	}

	// Remove agent node from sidebar
	removeAgentNodeFromSidebar(&m, msg.AgentID)
	return m, nil
}

func (m UIModel) handleAllStoppedAgentsCleared(msg AllStoppedAgentsClearedMsg) (tea.Model, tea.Cmd) {
	// Remove from agents map and clear view if needed
	for _, id := range msg.ClearedIDs {
		delete(m.agents, id)
		if m.viewingAgentID == id {
			m.viewingAgentID = ""
		}
	}

	// Rebuild sidebar to remove all cleared agents
	rebuildAgentNodesInSidebar(&m)
	return m, nil
}

func removeAgentNodeFromSidebar(m *UIModel, agentID string) {
	state := m.sidebar.State()
	for i := range state.Nodes {
		removeAgentFromProject(&state.Nodes[i], agentID)
	}
	state.RebuildFlatNodes()
}

func removeAgentFromProject(projectNode *domain.SidebarNode, agentID string) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree {
			// Filter out the agent with matching ID
			newChildren := make([]domain.SidebarNode, 0, len(worktreeNode.Children))
			for _, child := range worktreeNode.Children {
				if child.Type != domain.NodeTypeAgent || child.AgentInfo == nil || child.AgentInfo.ID != agentID {
					newChildren = append(newChildren, child)
				}
			}
			worktreeNode.Children = newChildren
		}
	}
}

func rebuildAgentNodesInSidebar(m *UIModel) {
	state := m.sidebar.State()
	for i := range state.Nodes {
		rebuildAgentsInProject(&state.Nodes[i], m.agents)
	}
	state.RebuildFlatNodes()
}

func rebuildAgentsInProject(projectNode *domain.SidebarNode, agents map[string]*RunningAgent) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree {
			// Keep only non-agent children and agents that still exist
			newChildren := make([]domain.SidebarNode, 0, len(worktreeNode.Children))
			for _, child := range worktreeNode.Children {
				if child.Type != domain.NodeTypeAgent {
					newChildren = append(newChildren, child)
				} else if child.AgentInfo != nil {
					// Check if agent still exists
					if _, ok := agents[child.AgentInfo.ID]; ok {
						newChildren = append(newChildren, child)
					}
				}
			}
			worktreeNode.Children = newChildren
		}
	}
}

// advanceFocus moves focus right, skipping disabled columns
func (m *UIModel) advanceFocus() {
	// Check if we can actually advance
	if m.focus >= FocusAgent {
		return
	}

	// Find the next enabled column
	for nextFocus := m.focus + 1; nextFocus <= FocusAgent; nextFocus++ {
		// Skip disabled columns
		if nextFocus == FocusModel && m.modelColumnDisabled {
			continue
		}
		if nextFocus == FocusAgent && m.agentColumnDisabled {
			continue
		}
		// Found an enabled column, move to it
		if m.focus == FocusSidebar {
			m.sidebar.SetFocused(false)
		}
		m.focus = nextFocus
		return
	}
	// No enabled column found, stay at current position
}

// retreatFocus moves focus left, skipping disabled columns
func (m *UIModel) retreatFocus() {
	// Check if we can actually retreat
	if m.focus <= FocusSidebar {
		return
	}

	// Find the previous enabled column
	for nextFocus := m.focus - 1; nextFocus >= FocusSidebar; nextFocus-- {
		// Skip disabled columns
		if nextFocus == FocusModel && m.modelColumnDisabled {
			continue
		}
		if nextFocus == FocusAgent && m.agentColumnDisabled {
			continue
		}
		// Found an enabled column, move to it
		if nextFocus == FocusSidebar {
			m.sidebar.SetFocused(true)
		}
		m.focus = nextFocus
		return
	}
	// No enabled column found, stay at current position
}

func (m UIModel) handleAnimationTick(msg animationTickMsg) (tea.Model, tea.Cmd) {
	elapsed := msg.Time.Sub(m.animState.StartTime).Seconds()

	// Pulse cycle: 0 to 1 to 0 over PulsePeriodSeconds
	// Using sine wave: sin(2π * t / period)
	period := PulsePeriodSeconds
	phase := (math.Sin(2*math.Pi*elapsed/period) + 1) / 2 // Normalize to 0-1

	m.animState.PulsePhase = phase

	// Decay lock-in flash intensity
	if m.animState.LockInActive {
		flashElapsed := msg.Time.Sub(m.animState.LockInStartTime).Milliseconds()
		flashDuration := int64(LockInFlashDuration / time.Millisecond)

		if flashElapsed >= flashDuration {
			// Flash complete - reset to inactive state
			m.animState.LockInActive = false
			m.animState.LockInIntensity = 0.0
		} else {
			// Linear decay: 1.0 → 0.0 over the flash duration
			m.animState.LockInIntensity = 1.0 - float64(flashElapsed)/float64(flashDuration)
		}
	}

	// Continue animation loop
	return m, animationTickCmd()
}

func createErrorList(message string) list.Model {
	items := []list.Item{errorItem{message: message}}
	l := list.New(items, newGradientDelegate(), 0, 0)
	l.Title = "Select a Ticket"
	l.SetShowStatusBar(false)
	return l
}

type errorItem struct {
	message string
}

func (i errorItem) Title() string       { return "⚠ " + i.message }
func (i errorItem) Description() string { return "" }
func (i errorItem) FilterValue() string { return "" }

func (m UIModel) handleTicketUpdateCheck() (tea.Model, tea.Cmd) {
	store := m.app.Store()
	if store == nil {
		return m, tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
			return ticketUpdateCheckMsg{}
		})
	}
	return m, checkTicketUpdatesCmd(store, m.lastTicketUpdate)
}

func (m UIModel) handleTicketsAutoRefreshed() (tea.Model, tea.Cmd) {
	m.refreshedRecently = true
	m.refreshAnimationFrame = 0

	cmds := []tea.Cmd{loadTicketsCmd(m.app.Store())}

	if m.app.Fonts.HasNerdFont {
		cmds = append(cmds, tea.Tick(animationTickInterval, func(t time.Time) tea.Msg {
			return refreshAnimationTickMsg{}
		}))
	}

	cmds = append(cmds, tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
		return clearRefreshIndicatorMsg{}
	}))

	return m, tea.Batch(cmds...)
}

func (m UIModel) handleClearRefreshIndicator() (tea.Model, tea.Cmd) {
	m.refreshedRecently = false
	return m, nil
}

func (m UIModel) handleRefreshAnimationTick() (tea.Model, tea.Cmd) {
	m.refreshAnimationFrame = (m.refreshAnimationFrame + 1) % 4
	return m, tea.Tick(animationTickInterval, func(t time.Time) tea.Msg {
		return refreshAnimationTickMsg{}
	})
}

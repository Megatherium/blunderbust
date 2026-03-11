package ui

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	_ "github.com/megatherium/blunderbust/internal/ui/filepicker"

	"github.com/megatherium/blunderbust/internal/data/dolt"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
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
	m.ticketList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.harnessList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.modelList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.agentList.SetDelegate(newGradientDelegate(m.currentTheme))
	m.dirtyTicket = true
	m.dirtyHarness = true
	m.dirtyModel = true
	m.dirtyAgent = true
	return m, nil, true
}

func (m UIModel) handleNavigationKeysMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.state != ViewStateMatrix {
		return m, nil, false
	}

	// Don't process navigation keys when a list is in filtering mode
	if m.isFocusedListFiltering() {
		return m, nil, false
	}

	switch msg.String() {
	case "left", "h":
		return m.handleLeftNavigation()
	case "right", "l":
		return m.handleRightNavigation()
	case "tab":
		if m.focus < FocusAgent {
			m.advanceFocus()
		} else {
			m.focus = FocusSidebar
			m.sidebar.SetFocused(true)
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m UIModel) handleLeftNavigation() (tea.Model, tea.Cmd, bool) {
	if m.focus == FocusSidebar {
		node := m.sidebar.State().CurrentNode()
		shouldCollapse := node != nil && len(node.Children) > 0 && node.IsExpanded
		if shouldCollapse {
			return m, nil, false // Let sidebar handle collapse
		}
	}
	if m.focus > FocusSidebar {
		m.retreatFocus()
		return m, nil, true
	}
	return m, nil, false
}

func (m UIModel) handleRightNavigation() (tea.Model, tea.Cmd, bool) {
	if m.focus == FocusSidebar {
		node := m.sidebar.State().CurrentNode()
		shouldExpand := node != nil && len(node.Children) > 0 && !node.IsExpanded
		if shouldExpand {
			return m, nil, false // Let sidebar handle expand
		}
	}
	if m.focus < FocusAgent {
		m.advanceFocus()
		return m, nil, true
	}
	return m, nil, false
}

func (m UIModel) handleTicketsLoaded(msg ticketsLoadedMsg) (tea.Model, tea.Cmd) {
	// Save current ticket selection before loading new tickets
	var prevTicketID string
	if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
		prevTicketID = i.ticket.ID
	}

	if len(msg) == 0 {
		if m.app.Project() == nil || m.app.Project().Store() == nil {
			m.ticketList = createErrorList("Couldn't load ticket list:\nStore initialization failed", m.currentTheme)
			m.sidebar.SetStoreError(true)
			if m.state == ViewStateLoading {
				m.state = ViewStateMatrix
			}
			return m, nil
		}
		m.ticketList = newEmptyTicketList(m.currentTheme)
		m.sidebar.SetStoreError(false)
	} else {
		m.ticketList = newTicketList(msg, m.currentTheme)
		m.sidebar.SetStoreError(false)
	}
	initList(&m.ticketList, 0, 0, "Select a Ticket")
	if m.state == ViewStateLoading {
		m.state = ViewStateMatrix
	}
	m.updateSizes()
	m.lastTicketUpdate = time.Now()
	m.dirtyTicket = true

	// Restore ticket selection and visual cursor position if it still exists.
	// When tickets reorder (for example after priority changes), Select() with the
	// new index moves the visible cursor to the same ticket again.
	// When tickets stay at the same index, Select() alone may not visibly move the
	// cursor, so m.selection.Ticket remains the source of truth for logical state.
	if prevTicketID != "" {
		foundIndex := -1
		for idx, ticket := range msg {
			if ticket.ID == prevTicketID {
				m.selection.Ticket = ticket
				foundIndex = idx
				break
			}
		}
		if foundIndex >= 0 {
			m.ticketList.Select(foundIndex)
		}
		// Clear selection if previously selected ticket no longer exists
		if foundIndex < 0 {
			m.selection.Ticket = domain.Ticket{}
		}
	}

	return m, nil
}

func (m UIModel) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	m.state = ViewStateError
	// Preserve the store for retry/start operations in error recovery UI
	if project := m.app.Project(); project != nil {
		m.retryStore = project.Store()
	}
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
		selection := m.selection
		if msg.spec != nil {
			selection = msg.spec.Selection
		}

		// Create agent info
		agentID := msg.res.WindowName
		agentInfo := &domain.AgentInfo{
			ID:           agentID,
			Name:         selection.Ticket.ID,
			WindowName:   msg.res.WindowName,
			WindowID:     msg.res.WindowID,
			WorktreePath: m.selectedWorktree,
			Status:       domain.AgentRunning,
			StartedAt:    time.Now(),
			TicketID:     selection.Ticket.ID,
			TicketTitle:  selection.Ticket.Title,
			HarnessName:  selection.Harness.Name,
			ModelName:    selection.Model,
			AgentName:    selection.Agent,
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
			saveRunningAgentCmd(m.app, msg.spec, msg.res, m.selectedWorktree),
		)
	}

	// Return to matrix even on error (user can see error in sidebar status)
	m.state = ViewStateMatrix
	return m, nil
}

func (m UIModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (UIModel, tea.Cmd) {
	m.layout = Compute(msg.Width, msg.Height, m.showSidebar)
	m.dirtyTicket = true
	m.dirtyHarness = true
	m.dirtyModel = true
	m.dirtyAgent = true
	return m, nil
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

	if model, cmd, handled := m.handleSidebarAgentKeysMsg(msg); handled {
		return model, cmd, true
	}

	return m, nil, false
}

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

func (m UIModel) handleAgentEnterKey() (tea.Model, tea.Cmd) {
	if i, ok := m.agentList.SelectedItem().(agentItem); ok {
		m.selection.Agent = i.name
		m.state = ViewStateConfirm
		return m, nil
	}
	return m, nil
}

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

func (m UIModel) handleModelSkip() (UIModel, tea.Cmd) {
	models := m.selection.Harness.SupportedModels

	var warnings []string
	expandedModels := make([]string, 0, len(models))
	for _, model := range models {
		switch {
		case strings.HasPrefix(model, discovery.PrefixProvider):
			providerID := strings.TrimPrefix(model, discovery.PrefixProvider)
			providerModels := m.app.Registry.GetModelsForProvider(providerID)
			if len(providerModels) == 0 {
				warnings = append(warnings, fmt.Sprintf("no models found for provider: %s (registry may not be loaded)", providerID))
			} else {
				expandedModels = append(expandedModels, providerModels...)
			}
		case model == discovery.KeywordDiscoverActive:
			activeModels := m.app.Registry.GetActiveModels()
			if len(activeModels) == 0 {
				warnings = append(warnings, "no active models found (check provider API keys and ensure registry is loaded)")
			} else {
				expandedModels = append(expandedModels, activeModels...)
			}
		default:
			expandedModels = append(expandedModels, model)
		}
	}

	var cmd tea.Cmd
	if len(warnings) > 0 {
		cmd = func() tea.Msg {
			return warningMsg{err: fmt.Errorf("%s", strings.Join(warnings, "; "))}
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

	// Save current model selection before regenerating list
	var prevModel string
	if item, ok := m.modelList.SelectedItem().(modelItem); ok {
		prevModel = item.name
	}

	m.modelColumnDisabled = len(models) == 0
	if m.modelColumnDisabled {
		m.selection.Model = ""
	}
	m.modelList = newModelList(models, m.currentTheme)
	m.updateSizes()
	m.dirtyModel = true

	// Restore model selection if it still exists in the new list
	// Note: We only set m.selection.Model here, not call m.modelList.Select().
	// This is because bubbles/list v0.10.3's Select() doesn't restore visual cursor
	// position when the same item remains selected - it only updates internal state.
	// The visual cursor will jump due to library limitations, but the logical selection
	// state is preserved correctly for downstream use.
	if prevModel != "" && !m.modelColumnDisabled {
		found := false
		for _, modelName := range models {
			if modelName == prevModel {
				m.selection.Model = prevModel
				found = true
				break
			}
		}
		// Clear selection if previously selected model no longer exists
		if !found {
			m.selection.Model = ""
		}
	}

	return m, cmd
}

func (m UIModel) handleAgentSkip() (UIModel, tea.Cmd) {
	agents := m.selection.Harness.SupportedAgents

	// Save current agent selection before regenerating list
	var prevAgent string
	if item, ok := m.agentList.SelectedItem().(agentItem); ok {
		prevAgent = item.name
	}

	m.agentColumnDisabled = len(agents) == 0
	if m.agentColumnDisabled {
		m.selection.Agent = ""
	}

	m.agentList = newAgentList(agents, m.currentTheme)
	m.updateSizes()
	m.dirtyAgent = true

	// Restore agent selection if it still exists in the new list
	// Note: We only set m.selection.Agent here, not call m.agentList.Select().
	// This is because bubbles/list v0.10.3's Select() doesn't restore visual cursor
	// position when the same item remains selected - it only updates internal state.
	// The visual cursor will jump due to library limitations, but the logical selection
	// state is preserved correctly for downstream use.
	if prevAgent != "" && !m.agentColumnDisabled {
		found := false
		for _, agentName := range agents {
			if agentName == prevAgent {
				m.selection.Agent = prevAgent
				found = true
				break
			}
		}
		// Clear selection if previously selected agent no longer exists
		if !found {
			m.selection.Agent = ""
		}
	}

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

func (m UIModel) handleWorktreesDiscovered(msg worktreesDiscoveredMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Worktree discovery: %v", msg.err))
		return m, nil
	}

	// Save current cursor node and selection before updating sidebar
	state := m.sidebar.State()
	prevNode := state.CurrentNode()
	prevSelectedPath := m.selectedWorktree

	// Update sidebar nodes (this rebuilds flat nodes and may change structure)
	m.sidebar, _ = m.sidebar.Update(SidebarNodesMsg{Nodes: msg.nodes})

	// Preserve current selection if it still exists in the updated nodes
	if prevSelectedPath != "" {
		// Check if the previously selected worktree still exists
		found := false
		for _, info := range m.sidebar.State().FlatNodes {
			if info.Node.Path == prevSelectedPath {
				found = true
				break
			}
		}
		if found {
			// Worktree still exists, preserve selection
			m.selectedWorktree = prevSelectedPath
			m.sidebar.SetSelectedPath(prevSelectedPath)
		} else if len(msg.nodes) > 0 && len(msg.nodes[0].Children) > 0 {
			// Current selection no longer exists, fall back to first available worktree
			initialPath := msg.nodes[0].Children[0].Path
			m.selectedWorktree = initialPath
			m.sidebar.SetSelectedPath(initialPath)
		}
	} else if len(msg.nodes) > 0 && len(msg.nodes[0].Children) > 0 {
		// No previous selection, use first worktree as default
		initialPath := msg.nodes[0].Children[0].Path
		m.selectedWorktree = initialPath
		m.sidebar.SetSelectedPath(initialPath)
	}

	// Try to restore cursor to the same node it was on before refresh
	if prevNode != nil {
		for i, info := range m.sidebar.State().FlatNodes {
			if info.Node.Path == prevNode.Path {
				// Found the same node, move cursor to its new position
				m.sidebar.State().Cursor = i
				break
			}
		}
	}

	for _, running := range m.agents {
		if running != nil && running.Info != nil {
			addAgentNodeToSidebar(&m, running.Info)
		}
	}

	return m, nil
}

func (m UIModel) handleRunningAgentsLoaded(msg runningAgentsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Running agents load: %v", msg.err))
		return m, nil
	}

	var cmds []tea.Cmd
	for _, persisted := range msg.agents {
		agentID := persistedAgentID(persisted)
		if existing, ok := m.agents[agentID]; ok && existing != nil {
			existing.Info.Status = domain.AgentRunning
			continue
		}

		info := &domain.AgentInfo{
			ID:           agentID,
			Name:         persisted.Ticket,
			WindowName:   persisted.WindowName,
			WorktreePath: persisted.WorktreePath,
			Status:       domain.AgentRunning,
			StartedAt:    persisted.StartedAt,
			TicketID:     persisted.Ticket,
			TicketTitle:  persisted.TicketTitle,
			HarnessName:  persisted.HarnessName,
			ModelName:    persisted.Model,
			AgentName:    persisted.Agent,
		}
		m.agents[agentID] = &RunningAgent{Info: info}
		addAgentNodeToSidebar(&m, info)

		if persisted.WindowName != "" {
			cmds = append(cmds,
				pollAgentStatusCmd(m.app, agentID, persisted.WindowName),
				startAgentMonitoringCmd(agentID),
			)
		}
	}

	if len(cmds) == 0 {
		return m, nil
	}
	return m, tea.Batch(cmds...)
}

func (m UIModel) handleWorktreeSelected(msg WorktreeSelectedMsg) (tea.Model, tea.Cmd) {
	m.selectedWorktree = msg.Path
	m.sidebar.SetSelectedPath(msg.Path)

	m.focus = FocusTickets
	m.sidebar.SetFocused(false)
	m.dirtyTicket = true
	return m, nil
}

// Agent management helpers

func addAgentNodeToSidebar(m *UIModel, agentInfo *domain.AgentInfo) {
	state := m.sidebar.State()
	prevPath := ""
	if node := state.CurrentNode(); node != nil {
		prevPath = node.Path
	}

	for i := range state.Nodes {
		addAgentToProject(&state.Nodes[i], agentInfo)
	}
	state.RebuildFlatNodes()
	restoreSidebarCursorByPath(state, prevPath)
}

func addAgentToProject(projectNode *domain.SidebarNode, agentInfo *domain.AgentInfo) {
	for i := range projectNode.Children {
		worktreeNode := &projectNode.Children[i]
		if worktreeNode.Type == domain.NodeTypeWorktree && worktreeNode.Path == agentInfo.WorktreePath {
			for _, child := range worktreeNode.Children {
				if child.Type == domain.NodeTypeAgent && child.AgentInfo != nil && child.AgentInfo.ID == agentInfo.ID {
					return
				}
			}
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

func persistedAgentID(a domain.PersistedRunningAgent) string {
	if a.WindowName != "" && a.PID > 0 {
		return fmt.Sprintf("%s:%d", a.WindowName, a.PID)
	}
	if a.WindowName != "" {
		return a.WindowName
	}
	if a.PID > 0 {
		return fmt.Sprintf("pid:%d", a.PID)
	}
	return fmt.Sprintf("agent:%d", a.ID)
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

func (m UIModel) handleAgentHovered(msg AgentHoveredMsg) (tea.Model, tea.Cmd) {
	m.hoveredAgentID = msg.AgentID
	return m, nil
}

func (m UIModel) handleAgentHoverEnded(msg AgentHoverEndedMsg) (tea.Model, tea.Cmd) {
	m.hoveredAgentID = ""
	return m, nil
}

func (m UIModel) handleAgentSelected(msg AgentSelectedMsg) (tea.Model, tea.Cmd) {
	m.state = ViewStateAgentOutput
	m.viewingAgentID = msg.AgentID
	m.hoveredAgentID = ""

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
	if m.hoveredAgentID == msg.AgentID {
		m.hoveredAgentID = ""
	}

	// If we were viewing this agent, stop viewing
	if m.state == ViewStateAgentOutput && m.viewingAgentID == msg.AgentID {
		m.viewingAgentID = ""
		m.state = ViewStateMatrix
	}

	// Remove agent node from sidebar
	removeAgentNodeFromSidebar(&m, msg.AgentID)
	return m, nil
}

func (m UIModel) handleAllStoppedAgentsCleared(msg AllStoppedAgentsClearedMsg) (tea.Model, tea.Cmd) {
	// Remove from agents map and clear view if needed
	for _, id := range msg.ClearedIDs {
		delete(m.agents, id)
		if m.state == ViewStateAgentOutput && m.viewingAgentID == id {
			m.viewingAgentID = ""
			m.state = ViewStateMatrix
		}
		if m.hoveredAgentID == id {
			m.hoveredAgentID = ""
		}
	}

	// Rebuild sidebar to remove all cleared agents
	rebuildAgentNodesInSidebar(&m)
	return m, nil
}

func removeAgentNodeFromSidebar(m *UIModel, agentID string) {
	state := m.sidebar.State()
	prevPath := ""
	if node := state.CurrentNode(); node != nil {
		prevPath = node.Path
	}

	for i := range state.Nodes {
		removeAgentFromProject(&state.Nodes[i], agentID)
	}
	state.RebuildFlatNodes()
	restoreSidebarCursorByPath(state, prevPath)
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
	prevPath := ""
	if node := state.CurrentNode(); node != nil {
		prevPath = node.Path
	}

	for i := range state.Nodes {
		rebuildAgentsInProject(&state.Nodes[i], m.agents)
	}
	state.RebuildFlatNodes()
	restoreSidebarCursorByPath(state, prevPath)
}

func restoreSidebarCursorByPath(state *domain.SidebarState, path string) {
	if path == "" {
		return
	}
	if state.SelectByPath(path) {
		return
	}
	if len(state.FlatNodes) == 0 {
		state.Cursor = 0
		return
	}
	if state.Cursor >= len(state.FlatNodes) {
		state.Cursor = len(state.FlatNodes) - 1
	}
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

	// Mark current focus column dirty
	m.markColumnDirty(m.focus)

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
		// Mark new focus column dirty
		m.markColumnDirty(m.focus)
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

	// Mark current focus column dirty
	m.markColumnDirty(m.focus)

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
		// Mark new focus column dirty
		m.markColumnDirty(m.focus)
		return
	}
	// No enabled column found, stay at current position
}

// markColumnDirty sets the appropriate dirty flag based on the given focus type
func (m *UIModel) markColumnDirty(focus FocusColumn) {
	switch focus {
	case FocusTickets:
		m.dirtyTicket = true
	case FocusHarness:
		m.dirtyHarness = true
	case FocusModel:
		m.dirtyModel = true
	case FocusAgent:
		m.dirtyAgent = true
	}
}

// markAllColumnsDirty sets all column dirty flags to true
func (m *UIModel) markAllColumnsDirty() {
	m.dirtyTicket = true
	m.dirtyHarness = true
	m.dirtyModel = true
	m.dirtyAgent = true
}

func (m UIModel) handleAnimationTick(msg animationTickMsg) (tea.Model, tea.Cmd) {
	elapsed := msg.Time.Sub(m.animState.StartTime).Seconds()

	// Pulse cycle: 0 to 1 to 0 over PulsePeriodSeconds
	// Using sine wave: sin(2π * t / period)
	period := PulsePeriodSeconds
	phase := (math.Sin(2*math.Pi*elapsed/period) + 1) / 2 // Normalize to 0-1

	m.animState.PulsePhase = phase

	// Handle color cycling - change palette every ColorCycleInterval
	cycleElapsed := msg.Time.Sub(m.animState.ColorCycleStart).Seconds()
	if cycleElapsed >= ColorCycleInterval.Seconds() {
		var cycleCount int
		if m.currentTheme == nil {
			cycleCount = len(MatrixThemeColorCycles)
		} else {
			switch m.currentTheme.Name {
			case CyberpunkTheme.Name:
				cycleCount = len(CyberpunkThemeColorCycles)
			case TokyoNightTheme.Name:
				cycleCount = len(TokyoNightThemeColorCycles)
			default:
				cycleCount = len(MatrixThemeColorCycles)
			}
		}
		if cycleCount < 1 {
			cycleCount = 1
		}
		m.animState.ColorCycleIndex = (m.animState.ColorCycleIndex + 1) % cycleCount
		m.animState.ColorCycleStart = msg.Time
	}

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

	if !m.animState.LockInActive {
		// Stop the animation loop to conserve CPU when idle
		return m, nil
	}

	// Continue animation loop
	return m, animationTickCmd()
}

func createErrorList(message string, theme ...*ThemePalette) list.Model {
	items := []list.Item{errorItem{message: message}}
	l := list.New(items, newGradientDelegate(theme...), 0, 0)
	l.Title = "Select a Ticket"
	l.SetShowTitle(false)
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
	if m.app.Project() == nil {
		return m, tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
			return ticketUpdateCheckMsg{}
		})
	}

	store := m.app.Project().Store()
	if store == nil {
		return m, tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
			return ticketUpdateCheckMsg{}
		})
	}
	return m, checkTicketUpdatesCmd(store, m.lastTicketUpdate)
}

func (m UIModel) handleTicketUpdateCheckNeeded() (tea.Model, tea.Cmd) {
	return m, tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
		return ticketUpdateCheckMsg{}
	})
}

func (m UIModel) handleTicketsAutoRefreshed(msg ticketsAutoRefreshedMsg) (tea.Model, tea.Cmd) {
	if !msg.dbUpdatedAt.IsZero() {
		m.lastTicketUpdate = msg.dbUpdatedAt
	}
	m.refreshedRecently = true
	m.refreshAnimationFrame = 0

	cmds := []tea.Cmd{loadTicketsCmd(m.app.Project().Store()), discoverWorktreesCmd(m.app)}

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

func (m UIModel) handleSidebarAgentKeysMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if m.focus != FocusSidebar {
		return m, nil, false
	}

	switch msg.String() {
	case "c":
		node := m.sidebar.State().CurrentNode()
		if node != nil && node.Type == domain.NodeTypeAgent && node.AgentInfo != nil {
			var capture *tmux.OutputCapture
			if agent, ok := m.agents[node.AgentInfo.ID]; ok {
				capture = agent.Capture
			}
			return m, clearAgentCmd(node.AgentInfo.ID, capture), true
		}
	case "C":
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

	return m, nil, false
}

func (m UIModel) handleAddProjectConfirmed(msg addProjectConfirmedMsg) (tea.Model, tea.Cmd) {
	projectDir := msg.path

	for _, project := range m.app.GetProjects() {
		if filepath.Clean(project.Dir) == filepath.Clean(projectDir) {
			m.warnings = append(m.warnings, fmt.Sprintf("Project already exists: %s", projectDir))
			m.state = ViewStateFilePicker
			return m, nil
		}
	}

	project := domain.Project{
		Dir:  projectDir,
		Name: filepath.Base(projectDir),
	}
	m.app.AddProject(project)

	ctx := context.Background()
	beadsDir := filepath.Join(projectDir, ".beads")
	store, err := m.app.CreateStore(ctx, beadsDir)
	if err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Failed to create store for %s: %v", projectDir, err))
		m.state = ViewStateFilePicker
		return m, nil
	}
	m.app.AddStore(projectDir, store)

	if err := m.app.SetActiveProject(ctx, projectDir); err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Failed to activate project %s: %v", projectDir, err))
		m.state = ViewStateFilePicker
		return m, nil
	}

	if err := m.app.SaveConfig(); err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Failed to save config: %v", err))
	}

	m.state = ViewStateMatrix
	m.pendingProjectPath = ""

	return m, tea.Batch(
		m.loadRegistryCmd(),
		m.continueInitAfterRegistry(),
		func() tea.Msg {
			return warningMsg{fmt.Errorf("added project: %s", projectDir)}
		},
	)
}

func (m UIModel) isFocusedListFiltering() bool {
	if m.state != ViewStateMatrix {
		return false
	}

	switch m.focus {
	case FocusTickets:
		return m.ticketList.FilterState() == list.Filtering
	case FocusHarness:
		return m.harnessList.FilterState() == list.Filtering
	case FocusModel:
		return m.modelList.FilterState() == list.Filtering
	case FocusAgent:
		return m.agentList.FilterState() == list.Filtering
	}
	return false
}

func updateListCaches(m *UIModel) UIModel {
	if m.dirtyTicket || m.ticketViewCache == "" {
		m.ticketViewCache = m.ticketList.View()
		m.dirtyTicket = false
	}
	if m.dirtyHarness || m.harnessViewCache == "" {
		m.harnessViewCache = m.harnessList.View()
		m.dirtyHarness = false
	}
	if m.dirtyModel || m.modelViewCache == "" {
		m.modelViewCache = m.modelList.View()
		m.dirtyModel = false
	}
	if m.dirtyAgent || m.agentViewCache == "" {
		m.agentViewCache = m.agentList.View()
		m.dirtyAgent = false
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

package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

func (m UIModel) handleTicketsLoaded(msg ticketsLoadedMsg) (tea.Model, tea.Cmd) {
	var prevTicketID string
	if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
		prevTicketID = i.ticket.ID
	}

	// Ensure we have a live ticketDelegate reference; create one if missing.
	if m.ticketDel == nil {
		m.ticketDel = newTicketDelegate(m.currentTheme)
	} else {
		m.ticketDel.applyTheme(m.currentTheme)
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
		items := []list.Item{emptyTicketItem{}}
		m.ticketList = list.New(items, m.ticketDel, 0, 0)
		m.ticketList.SetShowStatusBar(false)
		m.sidebar.SetStoreError(false)
	} else {
		items := make([]list.Item, 0, len(msg))
		for i := range msg {
			items = append(items, ticketItem{ticket: msg[i]})
		}
		m.ticketList = list.New(items, m.ticketDel, 0, 0)
		m.sidebar.SetStoreError(false)
	}
	initList(&m.ticketList, 0, 0, "Select a Ticket")
	if m.state == ViewStateLoading {
		m.state = ViewStateMatrix
	}
	m.updateSizes()
	m.lastTicketUpdate = time.Now()
	m.dirtyTicket = true

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
		if foundIndex < 0 {
			m.selection.Ticket = domain.Ticket{}
		}
	}

	return m, nil
}

func (m UIModel) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	m.err = msg.err
	m.state = ViewStateError
	if project := m.app.Project(); project != nil {
		m.retryStore = project.Store()
	}
	return m, nil
}

const maxWarnings = 50

func (m UIModel) handleWarningMsg(msg warningMsg) (tea.Model, tea.Cmd) {
	m.warnings = append(m.warnings, msg.err.Error())
	if len(m.warnings) > maxWarnings {
		m.warnings = m.warnings[len(m.warnings)-maxWarnings:]
	}
	if m.app != nil && m.app.Opts.Debug {
		fmt.Fprintf(os.Stderr, "[DEBUG][i29d] handleWarningMsg: warnings count=%d (latest: %s)\n", len(m.warnings), msg.err.Error())
	}
	return m, nil
}

func (m UIModel) handleLaunchResult(msg launchResultMsg) (tea.Model, tea.Cmd) {
	m.launchResult = msg.res
	m.err = msg.err

	if msg.err != nil {
		m.state = ViewStateError
		return m, nil
	}

	if msg.res != nil && msg.res.LauncherID != "" {
		selection := m.selection
		if msg.spec != nil {
			selection = msg.spec.Selection
		}

		agentID := msg.res.LauncherID
		agentInfo := &domain.AgentInfo{
			ID:           agentID,
			Name:         selection.Ticket.ID,
			LauncherID:   msg.res.LauncherID,
			WorktreePath: m.selectedWorktree,
			Status:       domain.AgentRunning,
			StartedAt:    time.Now(),
			TicketID:     selection.Ticket.ID,
			TicketTitle:  selection.Ticket.Title,
			HarnessName:  selection.Harness.Name,
			ModelName:    selection.Model,
			AgentName:    selection.Agent,
		}

		var capture *tmux.OutputCapture
		launcherID := msg.res.LauncherID
		if launcherID != "" && m.app.Runner() != nil && msg.res.LauncherType == domain.LauncherTypeTmux {
			capture = tmux.NewOutputCapture(m.app.Runner(), launcherID)
			path, captureErr := capture.Start(context.Background())
			if captureErr != nil {
				m.warnings = append(m.warnings, fmt.Sprintf("Failed to capture output: %v", captureErr))
				capture = nil
			}
			_ = path
		}

		m.agents[agentID] = &RunningAgent{
			Info:    agentInfo,
			Capture: capture,
		}

		AddAgentNodeToSidebar(&m, agentInfo)

		m.state = ViewStateMatrix

		return m, tea.Batch(
			pollAgentStatusCmd(m.app, agentID, msg.res.LauncherID),
			startAgentMonitoringCmd(agentID),
			saveRunningAgentCmd(m.app, msg.spec, msg.res, m.selectedWorktree),
		)
	}

	m.state = ViewStateMatrix
	return m, nil
}

func (m UIModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (UIModel, tea.Cmd) {
	m.layout = Compute(msg.Width, msg.Height, m.showSidebar)
	m.updateSizes()
	m.dirtyTicket = true
	m.dirtyHarness = true
	m.dirtyModel = true
	m.dirtyAgent = true
	return m, nil
}

func (m UIModel) handleWorktreesDiscovered(msg worktreesDiscoveredMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.warnings = append(m.warnings, fmt.Sprintf("Worktree discovery: %v", msg.err))
		return m, nil
	}

	state := m.sidebar.State()
	prevNode := state.CurrentNode()
	prevSelectedPath := m.selectedWorktree

	m.sidebar, _ = m.sidebar.Update(SidebarNodesMsg{Nodes: msg.nodes})

	if prevSelectedPath != "" {
		found := false
		for _, info := range m.sidebar.State().FlatNodes {
			if info.Node.Path == prevSelectedPath {
				found = true
				break
			}
		}
		if found {
			m.selectedWorktree = prevSelectedPath
			m.sidebar.SetSelectedPath(prevSelectedPath)
		} else if len(msg.nodes) > 0 && len(msg.nodes[0].Children) > 0 {
			initialPath := msg.nodes[0].Children[0].Path
			m.selectedWorktree = initialPath
			m.sidebar.SetSelectedPath(initialPath)
		}
	} else if len(msg.nodes) > 0 && len(msg.nodes[0].Children) > 0 {
		initialPath := msg.nodes[0].Children[0].Path
		m.selectedWorktree = initialPath
		m.sidebar.SetSelectedPath(initialPath)
	}

	if prevNode != nil {
		for i, info := range m.sidebar.State().FlatNodes {
			if info.Node.Path == prevNode.Path {
				m.sidebar.State().Cursor = i
				break
			}
		}
	}

	for _, running := range m.agents {
		if running != nil && running.Info != nil {
			AddAgentNodeToSidebar(&m, running.Info)
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
		agentID := PersistedAgentID(persisted)
		if existing, ok := m.agents[agentID]; ok && existing != nil {
			existing.Info.Status = domain.AgentRunning
			continue
		}

		info := &domain.AgentInfo{
			ID:           agentID,
			Name:         persisted.Ticket,
			LauncherID:   persisted.LauncherID,
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
		AddAgentNodeToSidebar(&m, info)

		if persisted.LauncherID != "" {
			cmds = append(cmds,
				pollAgentStatusCmd(m.app, agentID, persisted.LauncherID),
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

	cmds = append(cmds, tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
		return ticketUpdateCheckMsg{}
	}))

	return m, tea.Batch(cmds...)
}

func (m UIModel) handleClearRefreshIndicator() (tea.Model, tea.Cmd) {
	m.refreshedRecently = false
	return m, nil
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

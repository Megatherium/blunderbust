package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

func initList(l *list.Model, width, height int, title string) {
	l.Title = title
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	if width > 0 && height > 0 {
		l.SetSize(width, height)
	}
}

func NewUIModel(app *App, harnesses []domain.Harness) UIModel {
	currentTheme := &TokyoNightTheme

	hl := newHarnessList(harnesses, app.Registry, currentTheme)
	initList(&hl, 0, 0, "Select a Harness")

	tl := newTicketList(nil, currentTheme)
	initList(&tl, 0, 0, "Select a Ticket")

	ml := newModelList(nil, currentTheme)
	initList(&ml, 0, 0, "Select a Model")

	al := newAgentList(nil, currentTheme)
	initList(&al, 0, 0, "Select an Agent")

	h := help.New()
	h.ShowAll = false

	h.Styles.ShortKey = h.Styles.ShortKey.Background(ThemeFooterBg).Foreground(ThemeFooterFg).Bold(true)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Background(ThemeFooterBg).Foreground(ThemeFooterFg)
	h.Styles.ShortSeparator = h.Styles.ShortSeparator.Background(ThemeFooterBg).Foreground(ThemeFooterFg)

	return UIModel{
		app:          app,
		state:        ViewStateMatrix,
		focus:        FocusSidebar,
		harnesses:    harnesses,
		ticketList:   tl,
		harnessList:  hl,
		modelList:    ml,
		agentList:    al,
		sidebar:      NewSidebarModel(),
		help:         h,
		keys:         keys,
		loading:      true,
		showModal:    false,
		showSidebar:  true,
		agents:       make(map[string]*RunningAgent),
		currentTheme: currentTheme, // Default to TokyoNight theme
		animState: AnimationState{
			StartTime:       time.Now(),
			ColorCycleStart: time.Now(),
			CurrentThemeIdx: 2, // TokyoNight theme index (2)
		},
	}.initSidebar()
}

func (m UIModel) initSidebar() UIModel {
	m.sidebar.SetHasNerdFont(m.app.Fonts.HasNerdFont)
	return m
}

func (m UIModel) checkAndPromptAddProject(dirPath string) tea.Cmd {
	return func() tea.Msg {
		beadsPath := filepath.Join(dirPath, ".beads")
		if _, err := os.Stat(beadsPath); os.IsNotExist(err) {
			return errMsg{fmt.Errorf("no .beads directory found in %s", dirPath)}
		} else if err != nil {
			return errMsg{fmt.Errorf("error checking .beads directory: %w", err)}
		}
		return ShowAddProjectModalMsg{path: dirPath}
	}
}

func (m UIModel) Init() tea.Cmd {
	targetProject := m.app.GetTargetProject()
	if targetProject != "" {
		// Check if project is already in workspace
		if !m.app.IsProjectInWorkspace(targetProject) {
			// Validate the project has .beads directory
			if err := m.app.ValidateProject(targetProject); err != nil {
				// Show error modal for missing .beads
				return tea.Batch(
					func() tea.Msg {
						return errMsg{err}
					},
					m.loadRegistryCmd(),
				)
			}
			// Show add-project modal
			return tea.Batch(
				func() tea.Msg {
					return addProjectPromptMsg{projectPath: targetProject}
				},
				m.loadRegistryCmd(),
			)
		}
		// Project is in workspace, activate it
		if err := m.app.SetActiveProject(context.Background(), targetProject); err != nil {
			return tea.Batch(
				func() tea.Msg {
					return errMsg{err}
				},
				m.loadRegistryCmd(),
			)
		}
	}

	// Normal initialization flow - load registry first, then continue with tickets/worktrees
	return tea.Batch(
		m.loadRegistryCmd(),
	)
}

func (m UIModel) handleCoreMsgs(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case registryLoadedMsg:
		if len(m.harnesses) > 0 {
			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				m.selection.Harness = i.harness
				m, _ = m.handleModelSkip()
				m, _ = m.handleAgentSkip()
			}
		}
		return m, m.continueInitAfterRegistry(), true
	case ticketsLoadedMsg:
		m.lastTicketUpdate = latestTicketUpdate(msg)
		updatedM, _ := m.handleTicketsLoaded(msg)
		return updatedM, tea.Batch(
			tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
				return ticketUpdateCheckMsg{}
			}),
			loadRunningAgentsCmd(m.app),
		), true
	case errMsg:
		newM, cmd := m.handleErrMsg(msg)
		return newM, cmd, true
	case warningMsg:
		newM, cmd := m.handleWarningMsg(msg)
		return newM, cmd, true
	case modalContentMsg:
		m.modalContent = string(msg)
		return m, nil, true
	case tea.WindowSizeMsg:
		newM, cmd := m.handleWindowSizeMsg(msg)
		return newM, cmd, true
	case tea.KeyMsg:
		if model, cmd, handled := m.handleKeyMsg(msg); handled {
			return model, cmd, true
		}
	}
	return m, nil, false
}

func (m UIModel) handleProjectMsgs(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case addProjectPromptMsg:
		m.showAddProjectModal = true
		m.pendingProjectPath = msg.projectPath
		return m, nil, true
	case addProjectResultMsg:
		m.showAddProjectModal = false
		if msg.err != nil {
			m.err = msg.err
			m.state = ViewStateError
		} else if msg.success {
			return m, m.activateProjectAndInit(m.pendingProjectPath), true
		}
		return m, m.continueNormalInit(), true
	case worktreesDiscoveredMsg:
		newM, cmd := m.handleWorktreesDiscovered(msg)
		return newM, cmd, true
	case runningAgentsLoadedMsg:
		newM, cmd := m.handleRunningAgentsLoaded(msg)
		return newM, cmd, true
	case WorktreeSelectedMsg:
		newM, cmd := m.handleWorktreeSelected(msg)
		return newM, cmd, true
	case serverStartedMsg:
		activeProject := m.app.activeProject
		if activeProject != "" {
			m.app.stores[activeProject] = msg.store
		}
		return m, loadTicketsCmd(msg.store), true
	case OpenFilePickerMsg:
		m.showFilePicker = true
		m.showAddProjectModal = false
		m.pendingProjectPath = ""
		return m, nil, true
	case ShowAddProjectModalMsg:
		m.showFilePicker = false
		m.showAddProjectModal = true
		m.pendingProjectPath = msg.path
		return m, nil, true
	case addProjectConfirmedMsg:
		newM, cmd := m.handleAddProjectConfirmed(msg)
		return newM, cmd, true
	case addProjectCancelledMsg:
		m.showFilePicker = true
		m.showAddProjectModal = false
		m.pendingProjectPath = ""
		return m, nil, true
	}
	return m, nil, false
}

func (m UIModel) handleAgentMsgs(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case launchResultMsg:
		newM, cmd := m.handleLaunchResult(msg)
		return newM, cmd, true
	case AgentHoveredMsg:
		newM, cmd := m.handleAgentHovered(msg)
		return newM, cmd, true
	case AgentHoverEndedMsg:
		newM, cmd := m.handleAgentHoverEnded(msg)
		return newM, cmd, true
	case AgentSelectedMsg:
		newM, cmd := m.handleAgentSelected(msg)
		return newM, cmd, true
	case AgentStatusMsg:
		newM, cmd := m.handleAgentStatus(msg)
		return newM, cmd, true
	case agentTickMsg:
		newM, cmd := m.handleAgentTick(msg)
		return newM, cmd, true
	case agentOutputMsg:
		newM, cmd := m.handleAgentOutput(msg)
		return newM, cmd, true
	case animationTickMsg:
		newM, cmd := m.handleAnimationTick(msg)
		return newM, cmd, true
	case lockInMsg:
		m.animState.LockInActive = true
		m.animState.LockInIntensity = 1.0
		m.animState.LockInStartTime = time.Now()
		m.animState.LockInTarget = msg.Column
		return m, nil, true
	case AgentClearedMsg:
		newM, cmd := m.handleAgentCleared(msg)
		return newM, cmd, true
	case AllStoppedAgentsClearedMsg:
		newM, cmd := m.handleAllStoppedAgentsCleared(msg)
		return newM, cmd, true
	case ticketUpdateCheckMsg:
		newM, cmd := m.handleTicketUpdateCheck()
		return newM, cmd, true
	case ticketsAutoRefreshedMsg:
		newM, cmd := m.handleTicketsAutoRefreshed(msg)
		return newM, cmd, true
	case clearRefreshIndicatorMsg:
		newM, cmd := m.handleClearRefreshIndicator()
		return newM, cmd, true
	case refreshAnimationTickMsg:
		newM, cmd := m.handleRefreshAnimationTick()
		return newM, cmd, true
	}
	return m, nil, false
}

func (m UIModel) handleSidebarFocusUpdate(msg tea.Msg) (UIModel, tea.Cmd) {
	var cmd tea.Cmd
	prevCursor := m.sidebar.State().Cursor
	m.sidebar, cmd = m.sidebar.Update(msg)

	if m.sidebar.State().Cursor != prevCursor {
		node := m.sidebar.State().CurrentNode()
		if node != nil {
			var newProjectDir string
			if node.Type == domain.NodeTypeWorktree && node.ParentProject != nil {
				newProjectDir = node.ParentProject.Path
			} else if node.Type == domain.NodeTypeProject {
				newProjectDir = node.Path
			}

			if newProjectDir != "" && newProjectDir != m.app.activeProject {
				err := m.app.SetActiveProject(context.Background(), newProjectDir)
				if err == nil {
					m.selection.Ticket = domain.Ticket{}
					m.selection.Model = ""
					m.selection.Agent = ""
					cmd = tea.Batch(cmd, loadTicketsCmd(m.app.Project().Store()))
				}
			}
		}
	}
	return m, cmd
}

func (m UIModel) handleHarnessFocusUpdate(msg tea.Msg) (UIModel, tea.Cmd) {
	var cmd tea.Cmd
	var prevHarness string
	if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
		prevHarness = i.harness.Name
	}

	m.harnessList, cmd = m.harnessList.Update(msg)

	if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
		if prevHarness != i.harness.Name {
			m.selection.Harness = i.harness
			m, _ = m.handleModelSkip()
			m, _ = m.handleAgentSkip()
		}
	}
	return m, cmd
}

func (m UIModel) handleFocusUpdate(msg tea.Msg) (UIModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.state != ViewStateMatrix {
		return m, cmd
	}

	switch m.focus {
	case FocusSidebar:
		return m.handleSidebarFocusUpdate(msg)
	case FocusTickets:
		m.ticketList, cmd = m.ticketList.Update(msg)
	case FocusHarness:
		return m.handleHarnessFocusUpdate(msg)
	case FocusModel:
		m.modelList, cmd = m.modelList.Update(msg)
	case FocusAgent:
		m.agentList, cmd = m.agentList.Update(msg)
	}
	return m, cmd
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if newModel, cmd, handled := m.handleCoreMsgs(msg); handled {
		return newModel, cmd
	}
	if newModel, cmd, handled := m.handleProjectMsgs(msg); handled {
		return newModel, cmd
	}
	if newModel, cmd, handled := m.handleAgentMsgs(msg); handled {
		return newModel, cmd
	}

	m, cmd := m.handleFocusUpdate(msg)
	m.updateKeyBindings()
	return m, cmd
}

// activateProjectAndInit adds the project and initializes the TUI with it.
func (m UIModel) activateProjectAndInit(projectPath string) tea.Cmd {
	// Capture app reference explicitly to avoid confusion about what we're mutating
	app := m.app
	return tea.Batch(
		func() tea.Msg {
			// Add project to workspace
			project := domain.Project{
				Dir:  projectPath,
				Name: filepath.Base(projectPath),
			}
			app.AddProject(project)

			// Activate the project
			if err := app.SetActiveProject(context.Background(), projectPath); err != nil {
				return errMsg{err}
			}

			// Load tickets
			projectCtx := app.Project()
			if projectCtx == nil {
				return errMsg{fmt.Errorf("failed to get project context")}
			}

			tickets, err := projectCtx.Store().ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
		discoverWorktreesCmd(app),
	)
}

// continueNormalInit proceeds with normal TUI initialization.
func (m UIModel) continueNormalInit() tea.Cmd {
	return tea.Batch(
		m.loadRegistryCmd(),
	)
}

// continueInitAfterRegistry loads tickets and worktrees after registry is ready.
// This is called from registryLoadedMsg handler to ensure registry is loaded
// before any model discovery happens.
func (m UIModel) continueInitAfterRegistry() tea.Cmd {
	// Capture app reference explicitly for clarity
	app := m.app
	return tea.Batch(
		func() tea.Msg {
			project, err := app.CreateProjectContext(context.Background())
			if err != nil {
				return errMsg{err}
			}

			tickets, err := project.Store().ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
		discoverWorktreesCmd(app),
		animationTickCmd(),
	)
}

// loadRegistryCmd returns a command that loads the model registry.
// Extracted to avoid code duplication in Init().
func (m UIModel) loadRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.app.Registry.Load(context.Background()); err != nil {
			return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
		}
		return registryLoadedMsg{}
	}
}

func latestTicketUpdate(tickets ticketsLoadedMsg) time.Time {
	var latest time.Time
	for _, ticket := range tickets {
		if ticket.UpdatedAt.After(latest) {
			latest = ticket.UpdatedAt
		}
	}
	return latest
}

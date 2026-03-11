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
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/ui/filepicker"
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

	var registry *discovery.Registry
	if app != nil {
		registry = app.Registry
	}

	hl := newHarnessList(harnesses, registry, currentTheme)
	initList(&hl, 0, 0, "Select a Harness")

	tl := newTicketList(nil, currentTheme)
	initList(&tl, 0, 0, "Select a Ticket")

	ml := newModelList(nil, currentTheme)
	initList(&ml, 0, 0, "Select a Model")

	al := newAgentList(nil, currentTheme)
	initList(&al, 0, 0, "Select an Agent")

	h := help.New()
	h.Styles.ShortKey = h.Styles.ShortKey.Background(ThemeFooterBg).
		Foreground(ThemeFooterFg).
		Bold(true)
	h.Styles.ShortDesc = h.Styles.ShortDesc.Background(ThemeFooterBg).Foreground(ThemeFooterFg)
	h.Styles.ShortSeparator = h.Styles.ShortSeparator.Background(ThemeFooterBg).
		Foreground(ThemeFooterFg)

	var recents []string
	var maxRecents int
	if app != nil && app.loader != nil {
		if cfg, err := app.loader.Load(app.opts.ConfigPath); err == nil && cfg != nil {
			recents = cfg.FilePickerRecents
			maxRecents = cfg.FilePickerMaxRecents
		}
	}

	fp := filepicker.New()
	fp.ShowRecents = true
	fp.Recents = recents
	if maxRecents > 0 {
		fp.MaxRecents = maxRecents
	}
	if wd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = wd
	}

	return UIModel{
		app:          app,
		state:        ViewStateLoading,
		focus:        FocusSidebar,
		harnesses:    harnesses,
		ticketList:   tl,
		harnessList:  hl,
		modelList:    ml,
		agentList:    al,
		filepicker:   fp,
		sidebar:      NewSidebarModel(),
		help:         h,
		keys:         keys,
		showModal:    false,
		showSidebar:  true,
		agents:       make(map[string]*RunningAgent),
		currentTheme: currentTheme, // Default to TokyoNight theme

		dirtyTicket:  true, // Initial build needed
		dirtyHarness: true,
		dirtyModel:   true,
		dirtyAgent:   true,

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
					return ShowAddProjectModalMsg{path: targetProject}
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
		m.state = ViewStateFilePicker
		m.pendingProjectPath = ""
		return m, nil, true
	case ShowAddProjectModalMsg:
		m.state = ViewStateAddProjectModal
		m.pendingProjectPath = msg.path
		return m, nil, true
	case addProjectConfirmedMsg:
		newM, cmd := m.handleAddProjectConfirmed(msg)
		return newM, cmd, true
	case addProjectCancelledMsg:
		m.state = ViewStateFilePicker
		m.pendingProjectPath = ""
		return m, nil, true
	case filepicker.RecentsChangedMsg:
		// Save recents to config when they change
		if m.app != nil && m.app.loader != nil {
			cfg, err := m.app.loader.Load(m.app.opts.ConfigPath)
			if err == nil && cfg != nil {
				cfg.FilePickerRecents = msg.Recents
				if err := m.app.loader.Save(m.app.opts.ConfigPath, cfg); err != nil {
					// Log error to stderr
					fmt.Fprintf(os.Stderr, "Failed to save recents: %v\n", err)
				}
			}
		}
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
		return m, animationTickCmd(), true
	case AgentClearedMsg:
		newM, cmd := m.handleAgentCleared(msg)
		return newM, cmd, true
	case AllStoppedAgentsClearedMsg:
		newM, cmd := m.handleAllStoppedAgentsCleared(msg)
		return newM, cmd, true
	case ticketUpdateCheckMsg:
		newM, cmd := m.handleTicketUpdateCheck()
		return newM, cmd, true
	case ticketUpdateCheckNeededMsg:
		newM, cmd := m.handleTicketUpdateCheckNeeded()
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
					m.dirtyTicket = true
					m.dirtyModel = true
					m.dirtyAgent = true
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
	m.dirtyHarness = true

	if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
		if prevHarness != i.harness.Name {
			m.selection.Harness = i.harness
			m.dirtyModel = true
			m.dirtyAgent = true
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
		m.dirtyTicket = true
	case FocusHarness:
		return m.handleHarnessFocusUpdate(msg)
	case FocusModel:
		m.modelList, cmd = m.modelList.Update(msg)
		m.dirtyModel = true
	case FocusAgent:
		m.agentList, cmd = m.agentList.Update(msg)
		m.dirtyAgent = true
	}
	return m, cmd
}

// activateProjectAndInit adds the project and initializes the TUI with it.

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
		// Animation tick is only started on demand (LockIn) to save CPU
	)
}

// loadRegistryCmd returns a command that loads the model registry.
// Extracted to avoid code duplication in Init().
func (m UIModel) loadRegistryCmd() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return registryLoadedMsg{}
		}
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

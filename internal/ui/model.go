package ui

import (
	"context"
	"fmt"
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
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	if width > 0 && height > 0 {
		l.SetSize(width, height)
	}
}

func NewUIModel(app *App, harnesses []domain.Harness) UIModel {
	hl := newHarnessList(harnesses, app.Registry)
	initList(&hl, 0, 0, "Select a Harness")

	tl := newTicketList(nil)
	initList(&tl, 0, 0, "Select a Ticket")

	ml := newModelList(nil)
	initList(&ml, 0, 0, "Select a Model")

	al := newAgentList(nil)
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
		currentTheme: &TokyoNightTheme, // Default to TokyoNight theme
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
					func() tea.Msg {
						if err := m.app.Registry.Load(context.Background()); err != nil {
							return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
						}
						return registryLoadedMsg{}
					},
				)
			}
			// Show add-project modal
			return tea.Batch(
				func() tea.Msg {
					return addProjectPromptMsg{projectPath: targetProject}
				},
				func() tea.Msg {
					if err := m.app.Registry.Load(context.Background()); err != nil {
						return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
					}
					return registryLoadedMsg{}
				},
			)
		}
		// Project is in workspace, activate it
		if err := m.app.SetActiveProject(context.Background(), targetProject); err != nil {
			return tea.Batch(
				func() tea.Msg {
					return errMsg{err}
				},
				func() tea.Msg {
					if err := m.app.Registry.Load(context.Background()); err != nil {
						return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
					}
					return registryLoadedMsg{}
				},
			)
		}
	}

	// Normal initialization flow
	project, err := m.app.CreateProjectContext(context.Background())
	if err != nil {
		return tea.Batch(
			func() tea.Msg {
				if err := m.app.Registry.Load(context.Background()); err != nil {
					return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
				}
				return registryLoadedMsg{}
			},
			func() tea.Msg {
				return errMsg{err}
			},
			discoverWorktreesCmd(m.app),
			animationTickCmd(),
		)
	}

	return tea.Batch(
		func() tea.Msg {
			if err := m.app.Registry.Load(context.Background()); err != nil {
				return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
			}
			return registryLoadedMsg{}
		},
		func() tea.Msg {
			tickets, err := project.Store().ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
		discoverWorktreesCmd(m.app),
		animationTickCmd(),
		checkTicketUpdatesCmd(project.Store(), time.Time{}),
	)
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case registryLoadedMsg:
		if len(m.harnesses) > 0 {
			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				m.selection.Harness = i.harness
				m, _ = m.handleModelSkip()
				m, _ = m.handleAgentSkip()
			}
		}
		return m, nil

	case ticketsLoadedMsg:
		return m.handleTicketsLoaded(msg)

	case errMsg:
		return m.handleErrMsg(msg)

	case warningMsg:
		return m.handleWarningMsg(msg)

	case modalContentMsg:
		m.modalContent = string(msg)
		return m, nil

	case addProjectPromptMsg:
		m.showAddProjectModal = true
		m.pendingProjectPath = msg.projectPath
		return m, nil

	case addProjectResultMsg:
		m.showAddProjectModal = false
		if msg.err != nil {
			m.err = msg.err
			m.state = ViewStateError
		} else if msg.success {
			// Project added, now activate it and continue with normal init
			return m, m.activateProjectAndInit(m.pendingProjectPath)
		}
		// User declined, continue with normal init
		return m, m.continueNormalInit()

	case launchResultMsg:
		return m.handleLaunchResult(msg)

	case worktreesDiscoveredMsg:
		return m.handleWorktreesDiscovered(msg)

	case WorktreeSelectedMsg:
		return m.handleWorktreeSelected(msg)

	case AgentSelectedMsg:
		return m.handleAgentSelected(msg)

	case AgentStatusMsg:
		return m.handleAgentStatus(msg)

	case agentTickMsg:
		return m.handleAgentTick(msg)

	case agentOutputMsg:
		return m.handleAgentOutput(msg)

	case animationTickMsg:
		return m.handleAnimationTick(msg)

	case lockInMsg:
		// Trigger lock-in flash effect
		m.animState.LockInActive = true
		m.animState.LockInIntensity = 1.0
		m.animState.LockInStartTime = time.Now()
		m.animState.LockInTarget = msg.Column
		return m, nil

	case AgentClearedMsg:
		return m.handleAgentCleared(msg)

	case AllStoppedAgentsClearedMsg:
		return m.handleAllStoppedAgentsCleared(msg)

	case ticketUpdateCheckMsg:
		return m.handleTicketUpdateCheck()

	case ticketsAutoRefreshedMsg:
		return m.handleTicketsAutoRefreshed()

	case clearRefreshIndicatorMsg:
		return m.handleClearRefreshIndicator()

	case refreshAnimationTickMsg:
		return m.handleRefreshAnimationTick()

	case tea.WindowSizeMsg:
		m, cmd = m.handleWindowSizeMsg(msg)
		return m, cmd

	case tea.KeyMsg:
		if model, cmd, handled := m.handleKeyMsg(msg); handled {
			return model, cmd
		}
	}

	if m.state == ViewStateMatrix {
		switch m.focus {
		case FocusSidebar:
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
		case FocusTickets:
			m.ticketList, cmd = m.ticketList.Update(msg)
		case FocusHarness:
			var prevHarness string
			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				prevHarness = i.harness.Name
			}

			m.harnessList, cmd = m.harnessList.Update(msg)

			if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
				if prevHarness != i.harness.Name {
					// Harness selection changed, update downstream
					m.selection.Harness = i.harness
					m, _ = m.handleModelSkip()
					m, _ = m.handleAgentSkip()
				}
			}
		case FocusModel:
			m.modelList, cmd = m.modelList.Update(msg)
		case FocusAgent:
			m.agentList, cmd = m.agentList.Update(msg)
		}
	}

	m.updateKeyBindings()
	return m, cmd
}

// activateProjectAndInit adds the project and initializes the TUI with it.
func (m UIModel) activateProjectAndInit(projectPath string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			// Add project to workspace
			project := domain.Project{
				Dir:  projectPath,
				Name: filepath.Base(projectPath),
			}
			m.app.AddProject(project)

			// Activate the project
			if err := m.app.SetActiveProject(context.Background(), projectPath); err != nil {
				return errMsg{err}
			}

			// Load tickets
			projectCtx := m.app.Project()
			if projectCtx == nil {
				return errMsg{fmt.Errorf("failed to get project context")}
			}

			tickets, err := projectCtx.Store().ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
		discoverWorktreesCmd(m.app),
	)
}

// continueNormalInit proceeds with normal TUI initialization.
func (m UIModel) continueNormalInit() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			project, err := m.app.CreateProjectContext(context.Background())
			if err != nil {
				return errMsg{err}
			}

			tickets, err := project.Store().ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
		discoverWorktreesCmd(m.app),
	)
}

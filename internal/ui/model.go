package ui

import (
	"context"
	"fmt"
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
		app:         app,
		state:       ViewStateMatrix,
		focus:       FocusSidebar,
		harnesses:   harnesses,
		ticketList:  tl,
		harnessList: hl,
		modelList:   ml,
		agentList:   al,
		sidebar:     NewSidebarModel(),
		help:        h,
		keys:        keys,
		loading:     true,
		showModal:   false,
		showSidebar: true,
		agents:      make(map[string]*RunningAgent),
		animState: AnimationState{
			StartTime: time.Now(),
		},
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			if err := m.app.Registry.Load(context.Background()); err != nil {
				return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
			}
			return registryLoadedMsg{}
		},
		func() tea.Msg {
			store, err := m.app.CreateStore(context.Background())
			if err != nil {
				return errMsg{err}
			}
			tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
			if err != nil {
				return errMsg{err}
			}
			return ticketsLoadedMsg(tickets)
		},
		discoverWorktreesCmd(m.app.opts.BeadsDir),
		animationTickCmd(), // Start animation loop
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

	case AgentClearedMsg:
		return m.handleAgentCleared(msg)

	case AllStoppedAgentsClearedMsg:
		return m.handleAllStoppedAgentsCleared(msg)

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
			m.sidebar, cmd = m.sidebar.Update(msg)
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

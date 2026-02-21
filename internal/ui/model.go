package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbuss/internal/data"
	"github.com/megatherium/blunderbuss/internal/discovery"
	"github.com/megatherium/blunderbuss/internal/domain"
	"github.com/megatherium/blunderbuss/internal/exec/tmux"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Step int

const (
	StepTicketList Step = iota
	StepHarnessSelect
	StepModelSelect
	StepAgentSelect
	StepConfirm
	StepResult
	StepError
)

type UIModel struct {
	app       *App
	step      Step
	selection domain.Selection

	ticketList  list.Model
	harnessList list.Model
	modelList   list.Model
	agentList   list.Model

	harnesses []domain.Harness

	width  int
	height int

	err          error
	launchResult *domain.LaunchResult
	loading      bool

	// Window status monitoring
	windowStatus      string
	windowStatusEmoji string
	monitoringWindow  string
}

func initList(l *list.Model, width, height int, title string) {
	l.Title = title
	l.SetShowHelp(true)
	if width > 0 && height > 0 {
		l.SetSize(width, height)
	}
}

func NewUIModel(app *App, harnesses []domain.Harness) UIModel {
	hl := newHarnessList(harnesses)
	initList(&hl, 0, 0, "Select a Harness (esc: back)")

	tl := newTicketList(nil)
	initList(&tl, 0, 0, "Select a Ticket")

	ml := newModelList(nil)
	initList(&ml, 0, 0, "Select a Model (esc: back)")

	al := newAgentList(nil)
	initList(&al, 0, 0, "Select an Agent (esc: back)")

	return UIModel{
		app:         app,
		step:        StepTicketList,
		harnesses:   harnesses,
		ticketList:  tl,
		harnessList: hl,
		modelList:   ml,
		agentList:   al,
		loading:     true,
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			// Load model discovery registry in the background
			if err := m.app.Registry.Load(context.Background()); err != nil {
				// If load fails, we still continue but discovery might be empty
				if m.app.opts.Debug {
					// This is not a critical error, just log it
					return tea.Println(fmt.Sprintf("Non-critical error: Model discovery load failed: %v", err))
				}
			}
			return nil // No message needed on success
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
	)
}

type ticketsLoadedMsg []domain.Ticket
type errMsg struct{ err error }
type launchResultMsg struct {
	res *domain.LaunchResult
	err error
}
type statusUpdateMsg struct {
	status string
	emoji  string
}

func (e errMsg) Error() string { return e.err.Error() }

func loadTicketsCmd(store data.TicketStore) tea.Cmd {
	return func() tea.Msg {
		tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
		if err != nil {
			return errMsg{err}
		}
		return ticketsLoadedMsg(tickets)
	}
}

func (m UIModel) launchCmd() tea.Cmd {
	return func() tea.Msg {
		spec, err := m.app.Renderer.RenderSelection(m.selection)
		if err != nil {
			return launchResultMsg{res: nil, err: fmt.Errorf("failed to render launch spec: %w", err)}
		}

		// Set window name to ticket ID
		spec.WindowName = m.selection.Ticket.ID

		res, err := m.app.launcher.Launch(context.Background(), *spec)
		return launchResultMsg{res: res, err: err}
	}
}

// pollWindowStatusCmd creates a command that checks tmux window status
func (m UIModel) pollWindowStatusCmd(windowName string) tea.Cmd {
	return func() tea.Msg {
		if m.app.StatusChecker() == nil {
			return statusUpdateMsg{status: "Unknown", emoji: "⚪"}
		}

		status := m.app.StatusChecker().CheckStatus(context.Background(), windowName)
		var emoji string
		switch status {
		case tmux.Running:
			emoji = "🟢"
		case tmux.Dead:
			emoji = "🔴"
		default:
			emoji = "⚪"
		}

		return statusUpdateMsg{status: status.String(), emoji: emoji}
	}
}

// startMonitoringCmd starts the polling loop for window status
func (m UIModel) startMonitoringCmd(windowName string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		// This is a recurring tick, the actual check happens in Update
		return tickMsg{windowName: windowName}
	})
}

type tickMsg struct {
	windowName string
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ticketsLoadedMsg:
		return m.handleTicketsLoaded(msg)

	case errMsg:
		return m.handleErrMsg(msg)

	case launchResultMsg:
		return m.handleLaunchResult(msg)

	case statusUpdateMsg:
		return m.handleStatusUpdate(msg)

	case tickMsg:
		return m.handleTickMsg(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		if model, cmd, handled := m.handleKeyMsg(msg); handled {
			return model, cmd
		}
	}

	switch m.step {
	case StepTicketList:
		m.ticketList, cmd = m.ticketList.Update(msg)
	case StepHarnessSelect:
		m.harnessList, cmd = m.harnessList.Update(msg)
	case StepModelSelect:
		m.modelList, cmd = m.modelList.Update(msg)
	case StepAgentSelect:
		m.agentList, cmd = m.agentList.Update(msg)
	}
	return m, cmd
}

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
	m.step = StepError
	return m, nil
}

func (m UIModel) handleLaunchResult(msg launchResultMsg) (tea.Model, tea.Cmd) {
	m.launchResult = msg.res
	m.err = msg.err
	m.step = StepResult

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
	if m.step == StepResult && m.monitoringWindow == msg.windowName {
		return m, tea.Batch(
			m.pollWindowStatusCmd(msg.windowName),
			m.startMonitoringCmd(msg.windowName),
		)
	}
	return m, nil
}

func (m UIModel) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	h, v := docStyle.GetFrameSize()
	m.width, m.height = msg.Width-h, msg.Height-v
	m.ticketList.SetSize(m.width, m.height)
	m.harnessList.SetSize(m.width, m.height)
	m.modelList.SetSize(m.width, m.height)
	m.agentList.SetSize(m.width, m.height)
	return m, nil
}

func (m UIModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit, true
	case "r":
		if m.step == StepTicketList {
			m.loading = true
			return m, loadTicketsCmd(m.app.store), true
		}
	case "esc":
		if m.step > StepTicketList && m.step != StepResult && m.step != StepError {
			m.step--
			if m.step == StepAgentSelect && len(m.selection.Harness.SupportedAgents) <= 1 {
				m.step--
			}
			if m.step == StepModelSelect && len(m.selection.Harness.SupportedModels) <= 1 {
				m.step--
			}
			if m.step == StepHarnessSelect && len(m.harnesses) == 1 {
				m.step--
			}
			return m, nil, true
		}
	case "enter":
		model, cmd := m.handleEnterKey()
		return model, cmd, true
	}
	return m, nil, false
}

func (m UIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepTicketList:
		if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
			m.selection.Ticket = i.ticket
			if len(m.harnesses) == 1 {
				m.selection.Harness = m.harnesses[0]
				return m.handleModelSkip()
			}
			m.step = StepHarnessSelect
			return m, nil
		}
	case StepHarnessSelect:
		if i, ok := m.harnessList.SelectedItem().(harnessItem); ok {
			m.selection.Harness = i.harness
			m.step = StepModelSelect
			return m.handleModelSkip()
		}
	case StepModelSelect:
		if i, ok := m.modelList.SelectedItem().(modelItem); ok {
			m.selection.Model = i.name
			m.step = StepAgentSelect
			return m.handleAgentSkip()
		}
	case StepAgentSelect:
		if i, ok := m.agentList.SelectedItem().(agentItem); ok {
			m.selection.Agent = i.name
			m.step = StepConfirm
			return m, nil
		}
	case StepConfirm:
		m.step = StepResult
		return m, m.launchCmd()
	case StepResult:
		return m, tea.Quit
	}
	return m, nil
}

func (m UIModel) handleModelSkip() (tea.Model, tea.Cmd) {
	models := m.selection.Harness.SupportedModels

	// Expand providers if requested
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

	// Deduplicate
	uniqueModels := make([]string, 0, len(expandedModels))
	seen := make(map[string]bool)
	for _, model := range expandedModels {
		if !seen[model] {
			seen[model] = true
			uniqueModels = append(uniqueModels, model)
		}
	}
	models = uniqueModels

	if len(models) == 1 {
		m.selection.Model = models[0]
		m.step = StepAgentSelect
		return m.handleAgentSkip()
	} else if len(models) == 0 {
		m.selection.Model = ""
		m.step = StepAgentSelect
		return m.handleAgentSkip()
	}
	m.modelList = newModelList(models)
	initList(&m.modelList, m.width, m.height, "Select a Model (esc: back)")
	return m, nil
}

func (m UIModel) handleAgentSkip() (tea.Model, tea.Cmd) {
	agents := m.selection.Harness.SupportedAgents
	switch len(agents) {
	case 1:
		m.selection.Agent = agents[0]
		m.step = StepConfirm
	case 0:
		m.selection.Agent = ""
		m.step = StepConfirm
	default:
		m.agentList = newAgentList(agents)
		initList(&m.agentList, m.width, m.height, "Select an Agent (esc: back)")
	}
	return m, nil
}

func (m UIModel) View() string {
	var s string
	switch m.step {
	case StepTicketList:
		if m.loading {
			s = "Loading tickets..."
		} else {
			s = m.ticketList.View()
		}
	case StepHarnessSelect:
		s = m.harnessList.View()
	case StepModelSelect:
		s = m.modelList.View()
	case StepAgentSelect:
		s = m.agentList.View()
	case StepConfirm:
		s = confirmView(m.selection, m.app.Renderer, m.app.opts.DryRun)
	case StepResult:
		if m.launchResult == nil && m.err == nil {
			s = "Launching..."
		} else {
			s = resultView(m.launchResult, m.err, m.windowStatusEmoji, m.windowStatus)
		}
	case StepError:
		s = errorView(m.err)
	}

	return docStyle.Render(s)
}

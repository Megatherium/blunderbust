package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/megatherium/blunderbuss/internal/data"
	"github.com/megatherium/blunderbuss/internal/domain"
	"github.com/megatherium/blunderbuss/internal/exec"
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
	return func() tea.Msg {
		store, err := m.app.CreateStore(context.Background())
		if err != nil {
			return errMsg{err}
		}
		tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
		if err != nil {
			return errMsg{err}
		}
		return ticketsLoadedMsg(tickets)
	}
}

type ticketsLoadedMsg []domain.Ticket
type errMsg struct{ err error }
type launchResultMsg struct {
	res *domain.LaunchResult
	err error
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

func launchCmd(launcher exec.Launcher, selection domain.Selection) tea.Cmd {
	return func() tea.Msg {
		spec := domain.LaunchSpec{
			Selection:  selection,
			WindowName: fmt.Sprintf("dev-%s", selection.Ticket.ID),
		}
		res, err := launcher.Launch(context.Background(), spec)
		return launchResultMsg{res: res, err: err}
	}
}

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ticketsLoadedMsg:
		if len(msg) == 0 {
			// No tickets found - show empty state
			m.ticketList = newEmptyTicketList()
		} else {
			m.ticketList = newTicketList(msg)
		}
		initList(&m.ticketList, m.width, m.height, "Select a Ticket")
		m.loading = false
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.step = StepError
		return m, nil

	case launchResultMsg:
		m.launchResult = msg.res
		m.err = msg.err
		m.step = StepResult
		return m, nil

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.width, m.height = msg.Width-h, msg.Height-v
		m.ticketList.SetSize(m.width, m.height)
		m.harnessList.SetSize(m.width, m.height)
		m.modelList.SetSize(m.width, m.height)
		m.agentList.SetSize(m.width, m.height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
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

func (m UIModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "r":
		if m.step == StepTicketList {
			m.loading = true
			return m, loadTicketsCmd(m.app.store)
		}
	case "esc":
		if m.step > StepTicketList && m.step != StepResult && m.step != StepError {
			m.step--
			// Handle skip backwards tracking if we skipped lists
			if m.step == StepAgentSelect && len(m.selection.Harness.SupportedAgents) <= 1 {
				m.step--
			}
			if m.step == StepModelSelect && len(m.selection.Harness.SupportedModels) <= 1 {
				m.step--
			}
			if m.step == StepHarnessSelect && len(m.harnesses) == 1 {
				m.step--
			}
			return m, nil
		}
	case "enter":
		return m.handleEnterKey()
	}
	return m, nil
}

func (m UIModel) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepTicketList:
		if i, ok := m.ticketList.SelectedItem().(ticketItem); ok {
			m.selection.Ticket = i.ticket
			m.step = StepHarnessSelect
			if len(m.harnesses) == 1 {
				m.selection.Harness = m.harnesses[0]
				m.step = StepModelSelect
			}
			return m.handleModelSkip()
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
		return m, launchCmd(m.app.launcher, m.selection)
	case StepResult:
		return m, tea.Quit
	}
	return m, nil
}

func (m UIModel) handleModelSkip() (tea.Model, tea.Cmd) {
	models := m.selection.Harness.SupportedModels
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
		s = confirmView(m.selection, m.app.Renderer)
	case StepResult:
		if m.launchResult == nil && m.err == nil {
			s = "Launching..."
		} else {
			s = resultView(m.launchResult, m.err)
		}
	case StepError:
		s = errorView(m.err)
	}

	return docStyle.Render(s)
}

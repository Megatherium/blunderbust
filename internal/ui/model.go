package ui

import (
	"context"
	"fmt"

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
	hl := newHarnessList(harnesses)
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
		focus:       FocusTickets,
		harnesses:   harnesses,
		ticketList:  tl,
		harnessList: hl,
		modelList:   ml,
		agentList:   al,
		help:        h,
		keys:        keys,
		loading:     true,
		showModal:   false,
		showSidebar: true,
	}
}

func (m UIModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			if err := m.app.Registry.Load(context.Background()); err != nil {
				return warningMsg{err: fmt.Errorf("model discovery load failed: %w", err)}
			}
			return nil
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

func (m UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
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

	case statusUpdateMsg:
		return m.handleStatusUpdate(msg)

	case tickMsg:
		return m.handleTickMsg(msg)

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
		case FocusTickets:
			m.ticketList, cmd = m.ticketList.Update(msg)
		case FocusHarness:
			m.harnessList, cmd = m.harnessList.Update(msg)
		case FocusModel:
			m.modelList, cmd = m.modelList.Update(msg)
		case FocusAgent:
			m.agentList, cmd = m.agentList.Update(msg)
		}
	}

	m.updateKeyBindings()
	return m, cmd
}

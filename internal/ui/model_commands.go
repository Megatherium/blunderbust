package ui

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

func loadTicketsCmd(store data.TicketStore) tea.Cmd {
	return func() tea.Msg {
		tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
		if err != nil {
			return errMsg{err}
		}
		return ticketsLoadedMsg(tickets)
	}
}

func loadModalCmd(ticketID string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("bd", "show", ticketID).CombinedOutput()
		if err != nil {
			return modalContentMsg(fmt.Sprintf("Error loading bd show:\n%v\n%s", err, string(out)))
		}
		return modalContentMsg(string(out))
	}
}

func (m UIModel) launchCmd() tea.Cmd {
	return func() tea.Msg {
		spec, err := m.app.Renderer.RenderSelection(m.selection)
		if err != nil {
			return launchResultMsg{res: nil, err: fmt.Errorf("failed to render launch spec: %w", err)}
		}

		spec.WindowName = m.selection.Ticket.ID

		res, err := m.app.launcher.Launch(context.Background(), *spec)
		return launchResultMsg{res: res, err: err}
	}
}

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

func (m UIModel) startMonitoringCmd(windowName string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg{windowName: windowName}
	})
}

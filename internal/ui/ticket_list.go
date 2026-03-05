package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/megatherium/blunderbust/internal/domain"
)

type ticketItem struct {
	ticket domain.Ticket
}

func (i ticketItem) Title() string { return fmt.Sprintf("[%s] %s", i.ticket.ID, i.ticket.Title) }
func (i ticketItem) Description() string {
	return fmt.Sprintf("Status: %s | Priority: %d", i.ticket.Status, i.ticket.Priority)
}
func (i ticketItem) FilterValue() string { return i.ticket.Title }

func newTicketList(tickets []domain.Ticket, theme ...*ThemePalette) list.Model {
	items := make([]list.Item, 0, len(tickets))
	for i := range tickets {
		items = append(items, ticketItem{ticket: tickets[i]})
	}

	delegate := newGradientDelegate(theme...)
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a Ticket"
	l.SetShowTitle(false)
	return l
}

// emptyTicketItem represents an empty state message
type emptyTicketItem struct{}

func (i emptyTicketItem) Title() string       { return "No ready tickets found" }
func (i emptyTicketItem) Description() string { return "Press 'r' to refresh or 'q' to quit" }
func (i emptyTicketItem) FilterValue() string { return "" }

// newEmptyTicketList creates a list with a single empty state item
func newEmptyTicketList(theme ...*ThemePalette) list.Model {
	items := []list.Item{emptyTicketItem{}}
	l := list.New(items, newGradientDelegate(theme...), 0, 0)
	l.Title = "Select a Ticket"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	return l
}

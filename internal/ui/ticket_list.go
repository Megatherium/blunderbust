package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/megatherium/blunderbuss/internal/domain"
)

type ticketItem struct {
	ticket domain.Ticket
}

func (i ticketItem) Title() string { return fmt.Sprintf("[%s] %s", i.ticket.ID, i.ticket.Title) }
func (i ticketItem) Description() string {
	return fmt.Sprintf("Status: %s | Priority: %d", i.ticket.Status, i.ticket.Priority)
}
func (i ticketItem) FilterValue() string { return i.ticket.Title }

func newTicketList(tickets []domain.Ticket) list.Model {
	items := make([]list.Item, 0, len(tickets))
	for i := range tickets {
		items = append(items, ticketItem{ticket: tickets[i]})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a Ticket"
	return l
}

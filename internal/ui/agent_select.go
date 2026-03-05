package ui

import (
	"github.com/charmbracelet/bubbles/list"
)

type agentItem struct {
	name string
}

func (i agentItem) Title() string       { return i.name }
func (i agentItem) Description() string { return "AI Agent" }
func (i agentItem) FilterValue() string { return i.name }

func newAgentList(agents []string, theme ...*ThemePalette) list.Model {
	items := make([]list.Item, 0, len(agents))
	for _, a := range agents {
		items = append(items, agentItem{name: a})
	}

	delegate := newGradientDelegate(theme...)
	delegate.ShowDescription = false
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select an Agent"
	l.SetShowTitle(false)
	return l
}

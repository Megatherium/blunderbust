package ui

import (
	"github.com/charmbracelet/bubbles/list"
)

type modelItem struct {
	name string
}

func (i modelItem) Title() string       { return i.name }
func (i modelItem) Description() string { return "LLM Model" }
func (i modelItem) FilterValue() string { return i.name }

func newModelList(models []string) list.Model {
	var items []list.Item
	for _, m := range models {
		items = append(items, modelItem{name: m})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a Model"
	return l
}

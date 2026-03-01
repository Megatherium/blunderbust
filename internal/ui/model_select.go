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
	items := make([]list.Item, 0, len(models))
	for _, m := range models {
		items = append(items, modelItem{name: m})
	}

	delegate := newGradientDelegate()
	delegate.ShowDescription = false
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a Model"
	return l
}

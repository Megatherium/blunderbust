package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/megatherium/blunderbust/internal/domain"
)

type harnessItem struct {
	harness domain.Harness
}

func (i harnessItem) Title() string { return i.harness.Name }
func (i harnessItem) Description() string {
	return fmt.Sprintf("Models: %d\nAgents: %d", len(i.harness.SupportedModels), len(i.harness.SupportedAgents))
}
func (i harnessItem) FilterValue() string { return i.harness.Name }

func newHarnessList(harnesses []domain.Harness) list.Model {
	items := make([]list.Item, 0, len(harnesses))
	for i := range harnesses {
		items = append(items, harnessItem{harness: harnesses[i]})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(3) // Make room for multi-line description

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a Harness"
	return l
}

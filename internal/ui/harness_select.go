package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/megatherium/blunderbuss/internal/domain"
)

type harnessItem struct {
	harness domain.Harness
}

func (i harnessItem) Title() string { return i.harness.Name }
func (i harnessItem) Description() string {
	return fmt.Sprintf("Models: %d | Agents: %d", len(i.harness.SupportedModels), len(i.harness.SupportedAgents))
}
func (i harnessItem) FilterValue() string { return i.harness.Name }

func newHarnessList(harnesses []domain.Harness) list.Model {
	var items []list.Item
	for _, h := range harnesses {
		items = append(items, harnessItem{harness: h})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a Harness"
	return l
}

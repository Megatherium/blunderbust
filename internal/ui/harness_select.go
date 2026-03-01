package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
)

const (
	providerPrefix = "provider:"
	activeKeyword  = "discover:active"
)

type harnessItem struct {
	harness  domain.Harness
	registry *discovery.Registry
}

func (i harnessItem) Title() string { return i.harness.Name }

func (i harnessItem) Description() string {
	modelCount := i.getModelCount()
	return fmt.Sprintf("Models: %d\nAgents: %d", modelCount, len(i.harness.SupportedAgents))
}

func (i harnessItem) FilterValue() string { return i.harness.Name }

// getModelCount returns the actual number of resolved models.
// It expands provider wildcards (e.g., "provider:openai") and "discover:active"
// into actual model counts using the discovery registry.
//
// Note: "discover:active" expansion depends on environment variables being set
// for each provider (e.g., OPENAI_API_KEY, ANTHROPIC_API_KEY). The Registry
// checks these env vars to determine which providers are active.
func (i harnessItem) getModelCount() int {
	if i.harness.SupportedModels == nil {
		return 0
	}

	if i.registry == nil {
		// Fallback: just count raw entries if registry unavailable
		return len(i.harness.SupportedModels)
	}

	count := 0
	for _, model := range i.harness.SupportedModels {
		switch {
		case model == activeKeyword:
			// Expand to all active models from all providers
			activeModels := i.registry.GetActiveModels()
			count += len(activeModels)

		case strings.HasPrefix(model, providerPrefix):
			// Expand to models for this specific provider
			providerID := strings.TrimPrefix(model, providerPrefix)
			providerModels := i.registry.GetModelsForProvider(providerID)
			count += len(providerModels)

		default:
			// Regular model ID, count as 1
			count++
		}
	}

	return count
}

func newHarnessList(harnesses []domain.Harness, registry *discovery.Registry) list.Model {
	items := make([]list.Item, 0, len(harnesses))
	for i := range harnesses {
		items = append(items, harnessItem{
			harness:  harnesses[i],
			registry: registry,
		})
	}

	delegate := newGradientDelegate()
	// SetHeight(3) is required to prevent visual clipping of the 2-line description ("Models: X\nAgents: Y").
	// Default list delegates assume 1 line description (height 2 total).
	delegate.SetHeight(3)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Select a Harness"
	return l
}

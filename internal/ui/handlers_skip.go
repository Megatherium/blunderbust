package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/discovery"
)

// handleModelSkip regenerates the model list based on harness selection
// Expands provider: prefixes and handles discover:active keyword
func (m UIModel) handleModelSkip() (UIModel, tea.Cmd) {
	models := m.selection.Harness.SupportedModels

	var warnings []string
	expandedModels := make([]string, 0, len(models))
	for _, model := range models {
		switch {
		case strings.HasPrefix(model, discovery.PrefixProvider):
			providerID := strings.TrimPrefix(model, discovery.PrefixProvider)
			providerModels := m.app.Registry.GetModelsForProvider(providerID)
			if len(providerModels) == 0 {
				warnings = append(warnings, fmt.Sprintf("no models found for provider: %s (registry may not be loaded)", providerID))
			} else {
				expandedModels = append(expandedModels, providerModels...)
			}
		case model == discovery.KeywordDiscoverActive:
			activeModels := m.app.Registry.GetActiveModels()
			if len(activeModels) == 0 {
				warnings = append(warnings, "no active models found (check provider API keys and ensure registry is loaded)")
			} else {
				expandedModels = append(expandedModels, activeModels...)
			}
		default:
			expandedModels = append(expandedModels, model)
		}
	}

	var cmd tea.Cmd
	if len(warnings) > 0 {
		cmd = func() tea.Msg {
			return warningMsg{err: fmt.Errorf("%s", strings.Join(warnings, "; "))}
		}
	}

	uniqueModels := make([]string, 0, len(expandedModels))
	seen := make(map[string]bool)
	for _, model := range expandedModels {
		if !seen[model] {
			seen[model] = true
			uniqueModels = append(uniqueModels, model)
		}
	}
	models = uniqueModels

	// Save current model selection before regenerating list
	var prevModel string
	if item, ok := m.modelList.SelectedItem().(modelItem); ok {
		prevModel = item.name
	}

	m.modelColumnDisabled = len(models) == 0
	if m.modelColumnDisabled {
		m.selection.Model = ""
	}
	m.modelList = newModelList(models, m.currentTheme)
	m.updateSizes()
	m.dirtyModel = true

	// Restore model selection if it still exists in the new list
	// Note: We only set m.selection.Model here, not call m.modelList.Select().
	// This is because bubbles/list v0.10.3's Select() doesn't restore visual cursor
	// position when the same item remains selected - it only updates internal state.
	// The visual cursor will jump due to library limitations, but the logical selection
	// state is preserved correctly for downstream use.
	if prevModel != "" && !m.modelColumnDisabled {
		found := false
		for _, modelName := range models {
			if modelName == prevModel {
				m.selection.Model = prevModel
				found = true
				break
			}
		}
		// Clear selection if previously selected model no longer exists
		if !found {
			m.selection.Model = ""
		}
	}

	return m, cmd
}

// handleAgentSkip regenerates the agent list based on harness selection
// Preserves previous agent selection if still available
func (m UIModel) handleAgentSkip() (UIModel, tea.Cmd) {
	agents := m.selection.Harness.SupportedAgents

	// Save current agent selection before regenerating list
	var prevAgent string
	if item, ok := m.agentList.SelectedItem().(agentItem); ok {
		prevAgent = item.name
	}

	m.agentColumnDisabled = len(agents) == 0
	if m.agentColumnDisabled {
		m.selection.Agent = ""
	}

	m.agentList = newAgentList(agents, m.currentTheme)
	m.updateSizes()
	m.dirtyAgent = true

	// Restore agent selection if it still exists in the new list
	// Note: We only set m.selection.Agent here, not call m.agentList.Select().
	// This is because bubbles/list v0.10.3's Select() doesn't restore visual cursor
	// position when the same item remains selected - it only updates internal state.
	// The visual cursor will jump due to library limitations, but the logical selection
	// state is preserved correctly for downstream use.
	if prevAgent != "" && !m.agentColumnDisabled {
		found := false
		for _, agentName := range agents {
			if agentName == prevAgent {
				m.selection.Agent = prevAgent
				found = true
				break
			}
		}
		// Clear selection if previously selected agent no longer exists
		if !found {
			m.selection.Agent = ""
		}
	}

	return m, nil
}

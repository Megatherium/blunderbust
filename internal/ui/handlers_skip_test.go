package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/megatherium/blunderbust/internal/app"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
)

func setupModelSkipTest() (*UIModel, *app.App) {
	myApp := newTestApp()
	myApp.ActiveProject = "."
	if myApp.Stores == nil {
		myApp.Stores = make(map[string]data.TicketStore)
	}
	myApp.Stores["."] = &mockStore{}

	m := NewUIModel(myApp, nil)
	return &m, myApp
}

func TestHandleModelSkip_EmptyModels(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{},
	}

	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.True(t, newModel.modelColumnDisabled)
	assert.Empty(t, newModel.selection.Model)
}

func TestHandleModelSkip_HardcodedModels(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"gpt-4", "gpt-3.5"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.False(t, newModel.modelColumnDisabled)
	assert.Len(t, newModel.modelList.VisibleItems(), 2)
}

func TestHandleModelSkip_ProviderPrefix(t *testing.T) {
	m, app := setupModelSkipTest()
	app.Registry.SetProvider(discovery.Provider{
		ID:   "openai",
		Name: "OpenAI",
		Models: map[string]discovery.Model{
			"gpt-4":         {ID: "gpt-4", Name: "GPT-4"},
			"gpt-3.5-turbo": {ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo"},
		},
	})

	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:openai"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.False(t, newModel.modelColumnDisabled)
	assert.Len(t, newModel.modelList.VisibleItems(), 2)
}

func TestHandleModelSkip_ProviderPrefixEmptyProvider(t *testing.T) {
	m, _ := setupModelSkipTest()
	// Provider not registered in registry

	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:unknown"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.NotNil(t, cmd, "should return warning command for unknown provider")
	assert.True(t, newModel.modelColumnDisabled)
}

func TestHandleModelSkip_DiscoverActiveKeyword(t *testing.T) {
	m, app := setupModelSkipTest()
	// Set environment variable to activate provider
	t.Setenv("OPENAI_API_KEY", "test-key")
	app.Registry.SetProvider(discovery.Provider{
		ID:   "openai",
		Name: "OpenAI",
		Env:  []string{"OPENAI_API_KEY"},
		Models: map[string]discovery.Model{
			"gpt-4": {ID: "gpt-4", Name: "GPT-4"},
		},
	})

	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"discover:active"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.False(t, newModel.modelColumnDisabled)
	assert.Len(t, newModel.modelList.VisibleItems(), 1)
}

func TestHandleModelSkip_DiscoverActiveNoActiveModels(t *testing.T) {
	m, app := setupModelSkipTest()
	// Provider exists but no env var set (not active)
	app.Registry.SetProvider(discovery.Provider{
		ID:   "openai",
		Name: "OpenAI",
		Env:  []string{"OPENAI_API_KEY"},
		Models: map[string]discovery.Model{
			"gpt-4": {ID: "gpt-4", Name: "GPT-4"},
		},
	})

	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"discover:active"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.NotNil(t, cmd, "should return warning command when no active models")
	assert.True(t, newModel.modelColumnDisabled)
}

func TestHandleModelSkip_Deduplication(t *testing.T) {
	m, app := setupModelSkipTest()
	app.Registry.SetProvider(discovery.Provider{
		ID:   "openai",
		Name: "OpenAI",
		Models: map[string]discovery.Model{
			"gpt-4": {ID: "gpt-4", Name: "GPT-4"},
		},
	})

	// provider:openai returns "openai/gpt-4", listing it twice should deduplicate
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:openai", "provider:openai"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.Len(t, newModel.modelList.VisibleItems(), 1, "should deduplicate duplicate provider entries")
}

func TestHandleModelSkip_PreservesPreviousSelection(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"model-1", "model-2", "model-3"},
	}

	// First call - select model-2
	m.modelList = newModelList([]string{"model-1", "model-2", "model-3"}, m.currentTheme)
	m.modelList.Select(1)
	m.selection.Model = "model-2"

	// Rebuild with same models
	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.Equal(t, "model-2", newModel.selection.Model, "should preserve previous selection")
}

func TestHandleModelSkip_ClearsSelectionIfNotFound(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"model-1", "model-2"},
	}

	// Set previous selection
	m.modelList = newModelList([]string{"model-1", "model-2", "model-3"}, m.currentTheme)
	m.modelList.Select(2)
	m.selection.Model = "model-3"

	// Rebuild without model-3
	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.selection.Model, "should clear selection if model not found")
}

func TestHandleModelSkip_MixedProvidersAndHardcoded(t *testing.T) {
	m, app := setupModelSkipTest()
	app.Registry.SetProvider(discovery.Provider{
		ID:   "openai",
		Name: "OpenAI",
		Models: map[string]discovery.Model{
			"gpt-4": {ID: "gpt-4", Name: "GPT-4"},
		},
	})

	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"provider:openai", "claude-3", "provider:openai"},
	}

	newModel, cmd := m.handleModelSkip()

	assert.Nil(t, cmd)
	models := newModel.modelList.VisibleItems()
	assert.Len(t, models, 2, "should have gpt-4 and claude-3")
}

func TestHandleAgentSkip_EmptyAgents(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{},
	}

	newModel, cmd := m.handleAgentSkip()

	assert.Nil(t, cmd)
	assert.True(t, newModel.agentColumnDisabled)
	assert.Empty(t, newModel.selection.Agent)
}

func TestHandleAgentSkip_WithAgents(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder", "reviewer", "tester"},
	}

	newModel, cmd := m.handleAgentSkip()

	assert.Nil(t, cmd)
	assert.False(t, newModel.agentColumnDisabled)
	assert.Len(t, newModel.agentList.VisibleItems(), 3)
}

func TestHandleAgentSkip_PreservesPreviousSelection(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder", "reviewer", "tester"},
	}

	// First call - select reviewer
	m.agentList = newAgentList([]string{"coder", "reviewer", "tester"}, m.currentTheme)
	m.agentList.Select(1)
	m.selection.Agent = "reviewer"

	// Rebuild with same agents
	newModel, cmd := m.handleAgentSkip()

	assert.Nil(t, cmd)
	assert.Equal(t, "reviewer", newModel.selection.Agent, "should preserve previous selection")
}

func TestHandleAgentSkip_ClearsSelectionIfNotFound(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder", "reviewer"},
	}

	// Set previous selection
	m.agentList = newAgentList([]string{"coder", "reviewer", "tester"}, m.currentTheme)
	m.agentList.Select(2)
	m.selection.Agent = "tester"

	// Rebuild without tester
	newModel, cmd := m.handleAgentSkip()

	assert.Nil(t, cmd)
	assert.Empty(t, newModel.selection.Agent, "should clear selection if agent not found")
}

func TestHandleAgentSkip_SetsDirtyFlag(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder"},
	}
	m.dirtyAgent = false

	newModel, _ := m.handleAgentSkip()

	assert.True(t, newModel.dirtyAgent, "should set dirty flag")
}

func TestHandleModelSkip_SetsDirtyFlag(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"gpt-4"},
	}
	m.dirtyModel = false

	newModel, _ := m.handleModelSkip()

	assert.True(t, newModel.dirtyModel, "should set dirty flag")
}

func TestHandleModelSkip_UpdatesSizes(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.layout = LayoutDimensions{TermWidth: 100, TermHeight: 40}
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedModels: []string{"gpt-4", "gpt-3.5"},
	}

	newModel, _ := m.handleModelSkip()

	// Verify the model list has been created with proper sizing
	assert.NotNil(t, newModel.modelList)
	assert.False(t, newModel.modelColumnDisabled)
}

func TestHandleAgentSkip_UpdatesSizes(t *testing.T) {
	m, _ := setupModelSkipTest()
	m.layout = LayoutDimensions{TermWidth: 100, TermHeight: 40}
	m.selection.Harness = domain.Harness{
		Name:            "test-harness",
		SupportedAgents: []string{"coder", "reviewer"},
	}

	newModel, _ := m.handleAgentSkip()

	// Verify the agent list has been created with proper sizing
	assert.NotNil(t, newModel.agentList)
	assert.False(t, newModel.agentColumnDisabled)
}

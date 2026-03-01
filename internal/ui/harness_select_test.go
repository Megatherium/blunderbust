package ui

import (
	"os"
	"testing"

	"github.com/megatherium/blunderbust/internal/discovery"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestHarnessItem_getModelCount(t *testing.T) {
	tests := []struct {
		name     string
		models   []string
		setupEnv func()
		registry *discovery.Registry
		want     int
	}{
		{
			name:     "no registry - fallback to raw count",
			models:   []string{"openai/gpt-4", "anthropic/claude-3"},
			registry: nil,
			want:     2,
		},
		{
			name:     "regular models only - no wildcards",
			models:   []string{"openai/gpt-4", "anthropic/claude-3", "google/gemini-pro"},
			registry: setupMockRegistry(),
			want:     3,
		},
		{
			name:     "single provider wildcard",
			models:   []string{"provider:openai"},
			registry: setupMockRegistry(),
			want:     2, // openai has 2 models
		},
		{
			name:     "multiple provider wildcards",
			models:   []string{"provider:openai", "provider:anthropic"},
			registry: setupMockRegistry(),
			want:     4, // 2 + 2
		},
		{
			name:     "mixed regular and provider wildcards",
			models:   []string{"openai/gpt-4", "provider:anthropic"},
			registry: setupMockRegistry(),
			want:     3, // 1 + 2
		},
		{
			name:   "discover:active wildcard",
			models: []string{"discover:active"},
			setupEnv: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
				os.Setenv("ANTHROPIC_API_KEY", "test-key")
			},
			registry: setupMockRegistry(),
			want:     4, // all active models (2 from openai + 2 from anthropic)
		},
		{
			name:     "empty models list",
			models:   []string{},
			registry: setupMockRegistry(),
			want:     0,
		},
		{
			name:     "unknown provider wildcard",
			models:   []string{"provider:unknown"},
			registry: setupMockRegistry(),
			want:     0,
		},
		{
			name:   "complex mixed case",
			models: []string{"provider:openai", "custom/model", "discover:active"},
			setupEnv: func() {
				os.Setenv("OPENAI_API_KEY", "test-key")
				os.Setenv("ANTHROPIC_API_KEY", "test-key")
			},
			registry: setupMockRegistry(),
			want:     7, // 2 (openai) + 1 (custom) + 4 (all active)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
				defer os.Unsetenv("OPENAI_API_KEY")
				defer os.Unsetenv("ANTHROPIC_API_KEY")
			}
			item := harnessItem{
				harness: domain.Harness{
					Name:            "test-harness",
					SupportedModels: tt.models,
				},
				registry: tt.registry,
			}
			got := item.getModelCount()
			assert.Equal(t, tt.want, got, "getModelCount() mismatch")
		})
	}
}

func TestHarnessItem_Description(t *testing.T) {
	tests := []struct {
		name     string
		harness  domain.Harness
		registry *discovery.Registry
		want     string
	}{
		{
			name: "normal case with provider wildcard",
			harness: domain.Harness{
				Name:            "test-harness",
				SupportedModels: []string{"provider:openai"},
				SupportedAgents: []string{"agent1", "agent2"},
			},
			registry: setupMockRegistry(),
			want:     "Models: 2\nAgents: 2",
		},
		{
			name: "zero models",
			harness: domain.Harness{
				Name:            "test-harness",
				SupportedModels: []string{},
				SupportedAgents: []string{"agent1"},
			},
			registry: setupMockRegistry(),
			want:     "Models: 0\nAgents: 1",
		},
		{
			name: "zero agents",
			harness: domain.Harness{
				Name:            "test-harness",
				SupportedModels: []string{"openai/gpt-4"},
				SupportedAgents: []string{},
			},
			registry: setupMockRegistry(),
			want:     "Models: 1\nAgents: 0",
		},
		{
			name: "nil registry fallback",
			harness: domain.Harness{
				Name:            "test-harness",
				SupportedModels: []string{"model1", "model2", "model3"},
				SupportedAgents: []string{"agent1"},
			},
			registry: nil,
			want:     "Models: 3\nAgents: 1",
		},
		{
			name: "both zero with nil registry",
			harness: domain.Harness{
				Name:            "test-harness",
				SupportedModels: []string{},
				SupportedAgents: []string{},
			},
			registry: nil,
			want:     "Models: 0\nAgents: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := harnessItem{
				harness:  tt.harness,
				registry: tt.registry,
			}
			desc := item.Description()
			assert.Equal(t, tt.want, desc)
		})
	}
}

func setupMockRegistry() *discovery.Registry {
	registry, _ := discovery.NewRegistry("")
	registry.SetProviders(map[string]discovery.Provider{
		"openai": {
			ID:   "openai",
			Name: "OpenAI",
			Env:  []string{"OPENAI_API_KEY"}, // Set required env var for active detection
			Models: map[string]discovery.Model{
				"gpt-4":       {ID: "gpt-4", Name: "GPT-4"},
				"gpt-4-turbo": {ID: "gpt-4-turbo", Name: "GPT-4 Turbo"},
			},
		},
		"anthropic": {
			ID:   "anthropic",
			Name: "Anthropic",
			Env:  []string{"ANTHROPIC_API_KEY"}, // Set required env var for active detection
			Models: map[string]discovery.Model{
				"claude-3-opus":   {ID: "claude-3-opus", Name: "Claude 3 Opus"},
				"claude-3-sonnet": {ID: "claude-3-sonnet", Name: "Claude 3 Sonnet"},
			},
		},
	})
	return registry
}

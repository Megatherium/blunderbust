// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/megatherium/blunderbust/internal/domain"
)

func TestYAMLLoader_Load_ValidConfig(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode --model {{.Model}}"
    prompt_template: "Work on {{.TicketID}}"
    models:
      - claude-sonnet
      - o3
    agents:
      - coder
    env:
      LOG_LEVEL: debug
  - name: amp
    command_template: "amp"
    models: []
    agents: []
defaults:
  harness: opencode
  model: claude-sonnet
  agent: coder
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(config.Harnesses) != 2 {
		t.Errorf("Expected 2 harnesses, got %d", len(config.Harnesses))
	}

	// Check first harness
	opencode := config.Harnesses[0]
	if opencode.Name != "opencode" {
		t.Errorf("Expected name 'opencode', got %q", opencode.Name)
	}
	if opencode.CommandTemplate != "opencode --model {{.Model}}" {
		t.Errorf("Unexpected command_template: %q", opencode.CommandTemplate)
	}
	if opencode.PromptTemplate != "Work on {{.TicketID}}" {
		t.Errorf("Unexpected prompt_template: %q", opencode.PromptTemplate)
	}
	if len(opencode.SupportedModels) != 2 {
		t.Errorf("Expected 2 models, got %d", len(opencode.SupportedModels))
	}
	if len(opencode.SupportedAgents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(opencode.SupportedAgents))
	}
	if opencode.Env["LOG_LEVEL"] != "debug" {
		t.Errorf("Expected env LOG_LEVEL='debug', got %q", opencode.Env["LOG_LEVEL"])
	}

	// Check second harness (empty lists)
	amp := config.Harnesses[1]
	if amp.Name != "amp" {
		t.Errorf("Expected name 'amp', got %q", amp.Name)
	}
	if len(amp.SupportedModels) != 0 {
		t.Errorf("Expected empty models list, got %d items", len(amp.SupportedModels))
	}
	if len(amp.SupportedAgents) != 0 {
		t.Errorf("Expected empty agents list, got %d items", len(amp.SupportedAgents))
	}

	// Check defaults
	if config.Defaults == nil {
		t.Fatal("Expected defaults to be set")
	}
	if config.Defaults.Harness != "opencode" {
		t.Errorf("Expected default harness 'opencode', got %q", config.Defaults.Harness)
	}
}

func TestYAMLLoader_Load_MissingFile(t *testing.T) {
	loader := NewYAMLLoader()
	_, err := loader.Load("/nonexistent/path/config.yaml")

	if err == nil {
		t.Fatal("Expected error for missing file")
	}

	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("Error should mention 'config file not found', got: %v", err)
	}
}

func TestYAMLLoader_Load_InvalidYAML(t *testing.T) {
	yamlContent := `
harnesses:
  - name: test
    command_template: "test"
  - invalid yaml here: [}
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Error should mention 'failed to parse YAML', got: %v", err)
	}
}

func TestYAMLLoader_Load_MissingHarnessName(t *testing.T) {
	yamlContent := `
harnesses:
  - command_template: "test"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for missing harness name")
	}

	if !strings.Contains(err.Error(), "missing required field: name") {
		t.Errorf("Error should mention missing name field, got: %v", err)
	}
	if !strings.Contains(err.Error(), "index 0") {
		t.Errorf("Error should mention harness index, got: %v", err)
	}
}

func TestYAMLLoader_Load_MissingCommandTemplate(t *testing.T) {
	yamlContent := `
harnesses:
  - name: test-harness
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for missing command_template")
	}

	if !strings.Contains(err.Error(), "missing required field: command_template") {
		t.Errorf("Error should mention missing command_template, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test-harness") {
		t.Errorf("Error should mention harness name, got: %v", err)
	}
}

func TestYAMLLoader_Load_NoHarnesses(t *testing.T) {
	yamlContent := `
harnesses: []
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for empty harnesses list")
	}

	if !strings.Contains(err.Error(), "at least one harness") {
		t.Errorf("Error should mention needing at least one harness, got: %v", err)
	}
}

func TestYAMLLoader_Load_OptionalFieldsOmitted(t *testing.T) {
	yamlContent := `
harnesses:
  - name: minimal
    command_template: "minimal"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	harness := config.Harnesses[0]
	if harness.PromptTemplate != "" {
		t.Errorf("Expected empty prompt_template, got %q", harness.PromptTemplate)
	}
	if len(harness.SupportedModels) != 0 {
		t.Errorf("Expected empty models, got %d items", len(harness.SupportedModels))
	}
	if len(harness.SupportedAgents) != 0 {
		t.Errorf("Expected empty agents, got %d items", len(harness.SupportedAgents))
	}
	if len(harness.Env) != 0 {
		t.Errorf("Expected empty env, got %d items", len(harness.Env))
	}
}

func TestYAMLLoader_Load_MultipleValidationErrors(t *testing.T) {
	yamlContent := `
harnesses:
  - name: first
    command_template: "first"
  - name: second-bad
  - command_template: "third-bad"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error")
	}

	// Should fail on second harness (missing command_template)
	if !strings.Contains(err.Error(), "second-bad") {
		t.Errorf("Error should mention 'second-bad' harness, got: %v", err)
	}
}

func TestYAMLLoader_Load_FileBasedTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create template files
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	cmdTemplateContent := "opencode --model {{.Model}} --agent {{.Agent}} --debug"
	cmdTemplatePath := filepath.Join(templatesDir, "command.txt")
	if err := os.WriteFile(cmdTemplatePath, []byte(cmdTemplateContent), 0644); err != nil {
		t.Fatalf("Failed to write command template: %v", err)
	}

	promptTemplateContent := "Work on ticket {{.TicketID}}: {{.TicketTitle}}\n\n{{.TicketDescription}}"
	promptTemplatePath := filepath.Join(templatesDir, "prompt.txt")
	if err := os.WriteFile(promptTemplatePath, []byte(promptTemplateContent), 0644); err != nil {
		t.Fatalf("Failed to write prompt template: %v", err)
	}

	// Create config with file references
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "@./templates/command.txt"
    prompt_template: "@./templates/prompt.txt"
    models:
      - claude-sonnet
      - o3
    agents:
      - coder
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	opencode := config.Harnesses[0]
	if opencode.CommandTemplate != cmdTemplateContent {
		t.Errorf("Unexpected command_template: %q", opencode.CommandTemplate)
	}
	if opencode.PromptTemplate != promptTemplateContent {
		t.Errorf("Unexpected prompt_template: %q", opencode.PromptTemplate)
	}
}

func TestYAMLLoader_Load_FileBasedTemplates_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	yamlContent := `
harnesses:
  - name: test
    command_template: "@./templates/missing.txt"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for missing template file")
	}

	if !strings.Contains(err.Error(), "failed to load template file") {
		t.Errorf("Error should mention 'failed to load template file', got: %v", err)
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Error should mention 'file not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "@./templates/missing.txt") {
		t.Errorf("Error should include the original reference, got: %v", err)
	}
}

func TestYAMLLoader_Load_MixedInlineAndFileTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one template file
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	promptTemplateContent := "Complex prompt: {{.TicketID}}"
	promptTemplatePath := filepath.Join(templatesDir, "prompt.txt")
	if err := os.WriteFile(promptTemplatePath, []byte(promptTemplateContent), 0644); err != nil {
		t.Fatalf("Failed to write prompt template: %v", err)
	}

	yamlContent := `
harnesses:
  - name: inline-cmd
    command_template: "inline command {{.Model}}"
    prompt_template: "@./templates/prompt.txt"
  - name: file-cmd
    command_template: "@./templates/prompt.txt"
    prompt_template: "inline prompt {{.TicketID}}"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	inlineCmd := config.Harnesses[0]
	if inlineCmd.CommandTemplate != "inline command {{.Model}}" {
		t.Errorf("Unexpected inline command_template: %q", inlineCmd.CommandTemplate)
	}
	if inlineCmd.PromptTemplate != promptTemplateContent {
		t.Errorf("Unexpected file-based prompt_template: %q", inlineCmd.PromptTemplate)
	}

	fileCmd := config.Harnesses[1]
	if fileCmd.CommandTemplate != promptTemplateContent {
		t.Errorf("Unexpected file-based command_template: %q", fileCmd.CommandTemplate)
	}
	if fileCmd.PromptTemplate != "inline prompt {{.TicketID}}" {
		t.Errorf("Unexpected inline prompt_template: %q", fileCmd.PromptTemplate)
	}
}

func TestYAMLLoader_Load_FileBasedTemplates_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	emptyPath := filepath.Join(templatesDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to write empty template: %v", err)
	}

	yamlContent := `
harnesses:
  - name: test
    command_template: "@./templates/empty.txt"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for empty command template (required field)")
	}

	if !strings.Contains(err.Error(), "missing required field: command_template") {
		t.Errorf("Error should mention missing required field, got: %v", err)
	}
}

func TestYAMLLoader_Load_FileBasedTemplates_Subdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested template directory
	nestedDir := filepath.Join(tmpDir, "nested", "templates")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested templates dir: %v", err)
	}

	cmdContent := "nested command {{.Model}}"
	cmdPath := filepath.Join(nestedDir, "cmd.txt")
	if err := os.WriteFile(cmdPath, []byte(cmdContent), 0644); err != nil {
		t.Fatalf("Failed to write nested command template: %v", err)
	}

	yamlContent := `
harnesses:
  - name: test
    command_template: "@./nested/templates/cmd.txt"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Harnesses[0].CommandTemplate != cmdContent {
		t.Errorf("Unexpected command_template: %q", config.Harnesses[0].CommandTemplate)
	}
}

func TestYAMLLoader_Load_FileBasedTemplates_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a template file in a known location (using /tmp for simplicity)
	absTemplatesDir := t.TempDir()
	cmdContent := "absolute path command {{.TicketID}}"
	cmdPath := filepath.Join(absTemplatesDir, "cmd.txt")
	if err := os.WriteFile(cmdPath, []byte(cmdContent), 0644); err != nil {
		t.Fatalf("Failed to write absolute path template: %v", err)
	}

	// Config directory is different from template directory
	yamlContent := fmt.Sprintf(`
harnesses:
  - name: test
    command_template: "@%s"
`, cmdPath)
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Harnesses[0].CommandTemplate != cmdContent {
		t.Errorf("Unexpected command_template: %q", config.Harnesses[0].CommandTemplate)
	}
}

func TestYAMLLoader_Load_DuplicateHarnessNames(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode --model {{.Model}}"
  - name: amp
    command_template: "amp"
  - name: opencode
    command_template: "opencode --model {{.Model}}"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for duplicate harness name")
	}

	if !strings.Contains(err.Error(), "duplicate harness name") {
		t.Errorf("Error should mention 'duplicate harness name', got: %v", err)
	}
	if !strings.Contains(err.Error(), "\"opencode\"") {
		t.Errorf("Error should mention the duplicate name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "index 2") {
		t.Errorf("Error should mention duplicate at index 2, got: %v", err)
	}
	if !strings.Contains(err.Error(), "first defined at index 0") {
		t.Errorf("Error should mention first index, got: %v", err)
	}
}

func TestYAMLLoader_InterfaceCompliance(t *testing.T) {
	// This test ensures YAMLLoader implements Loader interface
	var _ Loader = (*YAMLLoader)(nil)
}

func TestYAMLLoader_Load_LauncherConfig_Foreground(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode"
launcher:
  target: foreground
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Launcher == nil {
		t.Fatal("Expected launcher config to be set")
	}
	if config.Launcher.Target != "foreground" {
		t.Errorf("Expected target 'foreground', got %q", config.Launcher.Target)
	}
}

func TestYAMLLoader_Load_LauncherConfig_Background(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode"
launcher:
  target: background
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Launcher == nil {
		t.Fatal("Expected launcher config to be set")
	}
	if config.Launcher.Target != "background" {
		t.Errorf("Expected target 'background', got %q", config.Launcher.Target)
	}
}

func TestYAMLLoader_Load_LauncherConfig_CaseInsensitive(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode"
launcher:
  target: BACKGROUND
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Launcher == nil {
		t.Fatal("Expected launcher config to be set")
	}
	if config.Launcher.Target != "background" {
		t.Errorf("Expected target 'background' (normalized), got %q", config.Launcher.Target)
	}
}

func TestYAMLLoader_Load_LauncherConfig_InvalidTarget(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode"
launcher:
  target: invalid
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("Expected error for invalid target value")
	}

	if !strings.Contains(err.Error(), "invalid launcher.target value") {
		t.Errorf("Error should mention invalid target, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Error should include the invalid value, got: %v", err)
	}
}

func TestYAMLLoader_Load_LauncherConfig_Missing(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Launcher == nil {
		t.Fatal("Expected launcher config to be set with default values")
	}
	if config.Launcher.Target != "foreground" {
		t.Errorf("Expected default target 'foreground', got %q", config.Launcher.Target)
	}
}

func TestYAMLLoader_Load_LauncherConfig_EmptyTarget(t *testing.T) {
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode"
launcher:
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if config.Launcher == nil {
		t.Fatal("Expected launcher config to be set")
	}
	if config.Launcher.Target != "foreground" {
		t.Errorf("Expected default target 'foreground', got %q", config.Launcher.Target)
	}
}

func TestYAMLLoader_Load_CompleteConfig(t *testing.T) {
	now := time.Now()
	yamlContent := `
harnesses:
  - name: opencode
    command_template: "opencode --model {{.Model}} --agent {{.Agent}}"
    prompt_template: "Work on ticket {{.TicketID}}: {{.TicketTitle}}"
    models:
      - claude-sonnet-4-20250514
      - o3
    agents:
      - coder
      - task
    env:
      KEY1: value1
      KEY2: value2
  - name: amp
    command_template: "amp"
    prompt_template: "Pick up {{.TicketID}}"
    models:
      - claude-sonnet-4-20250514
    agents: []
  - name: claude-code
    command_template: "claude"
    models: []
    agents: []
defaults:
  harness: opencode
  model: claude-sonnet-4-20250514
  agent: coder
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "blunderbust.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	loader := NewYAMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all 3 harnesses
	if len(config.Harnesses) != 3 {
		t.Fatalf("Expected 3 harnesses, got %d", len(config.Harnesses))
	}

	// Verify defaults
	if config.Defaults == nil {
		t.Fatal("Defaults should be populated")
	}

	// Create renderer and test full flow
	renderer := NewRenderer()
	selection := domain.Selection{
		Ticket: domain.Ticket{
			ID:          "bb-123",
			Title:       "Test Ticket",
			Description: "Test Description",
			Status:      "open",
			Priority:    1,
			IssueType:   "task",
			Assignee:    "testuser",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Harness: config.Harnesses[0],
		Model:   "claude-sonnet-4-20250514",
		Agent:   "coder",
	}

	spec, err := renderer.RenderSelection(selection, "")
	if err != nil {
		t.Fatalf("Failed to render selection: %v", err)
	}

	expectedCmd := "opencode --model claude-sonnet-4-20250514 --agent coder"
	if spec.RenderedCommand != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, spec.RenderedCommand)
	}

	expectedPrompt := "Work on ticket bb-123: Test Ticket"
	if spec.RenderedPrompt != expectedPrompt {
		t.Errorf("Expected prompt %q, got %q", expectedPrompt, spec.RenderedPrompt)
	}
}

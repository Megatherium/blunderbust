// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"strings"
	"testing"
	"time"

	"github.com/megatherium/blunderbuss/internal/domain"
)

func TestRenderer_RenderCommand_Simple(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:            "test",
		CommandTemplate: "echo {{.Model}}",
	}
	ctx := domain.TemplateContext{
		Model: "claude-sonnet",
	}

	result, err := renderer.RenderCommand(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "echo claude-sonnet"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderer_RenderCommand_Complex(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:            "opencode",
		CommandTemplate: "opencode --model {{.Model}} --agent {{.Agent}} --ticket {{.TicketID}}",
	}
	ctx := domain.TemplateContext{
		TicketID: "bb-abc",
		Model:    "claude-sonnet-4-20250514",
		Agent:    "coder",
	}

	result, err := renderer.RenderCommand(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "opencode --model claude-sonnet-4-20250514 --agent coder --ticket bb-abc"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderer_RenderCommand_InvalidTemplate(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:            "bad",
		CommandTemplate: "echo {{.BadField", // Invalid template syntax
	}
	ctx := domain.TemplateContext{}

	_, err := renderer.RenderCommand(harness, ctx)
	if err == nil {
		t.Fatal("Expected error for invalid template")
	}

	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("Error should mention harness name 'bad', got: %v", err)
	}
	if !strings.Contains(err.Error(), "command_template") {
		t.Errorf("Error should mention template type, got: %v", err)
	}
}

func TestRenderer_RenderCommand_MissingField(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:            "test",
		CommandTemplate: "echo {{.NonExistent}}",
	}
	ctx := domain.TemplateContext{}

	_, err := renderer.RenderCommand(harness, ctx)
	if err == nil {
		t.Fatal("Expected error for missing field in template execution")
	}

	if !strings.Contains(err.Error(), "test") {
		t.Errorf("Error should mention harness name, got: %v", err)
	}
}

func TestRenderer_RenderPrompt_WithTemplate(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:           "test",
		PromptTemplate: "Work on {{.TicketID}}: {{.TicketTitle}}",
	}
	ctx := domain.TemplateContext{
		TicketID:    "bb-123",
		TicketTitle: "Fix Bug",
	}

	result, err := renderer.RenderPrompt(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "Work on bb-123: Fix Bug"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderer_RenderPrompt_EmptyTemplate(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:           "minimal",
		PromptTemplate: "", // Empty template
	}
	ctx := domain.TemplateContext{
		TicketID: "bb-123",
	}

	result, err := renderer.RenderPrompt(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty string for empty template, got %q", result)
	}
}

func TestRenderer_RenderPrompt_NoTemplateField(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name: "no-prompt",
		// PromptTemplate not set (zero value)
	}
	ctx := domain.TemplateContext{
		TicketID: "bb-123",
	}

	result, err := renderer.RenderPrompt(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty string when no prompt_template, got %q", result)
	}
}

func TestRenderer_RenderPrompt_InvalidTemplate(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:           "bad-prompt",
		PromptTemplate: "{{.Bad", // Invalid syntax
	}
	ctx := domain.TemplateContext{}

	_, err := renderer.RenderPrompt(harness, ctx)
	if err == nil {
		t.Fatal("Expected error for invalid prompt template")
	}

	if !strings.Contains(err.Error(), "bad-prompt") {
		t.Errorf("Error should mention harness name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "prompt_template") {
		t.Errorf("Error should mention template type, got: %v", err)
	}
}

func TestRenderer_RenderSelection_Complete(t *testing.T) {
	renderer := NewRenderer()
	now := time.Now()

	selection := domain.Selection{
		Ticket: domain.Ticket{
			ID:          "bb-test",
			Title:       "Test Ticket",
			Description: "A test description",
			Status:      "open",
			Priority:    1,
			IssueType:   "task",
			Assignee:    "testuser",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Harness: domain.Harness{
			Name:            "opencode",
			CommandTemplate: "opencode --model {{.Model}} --agent {{.Agent}}",
			PromptTemplate:  "{{.TicketID}}: {{.TicketTitle}}\n{{.TicketDescription}}",
		},
		Model: "claude-sonnet",
		Agent: "coder",
	}

	spec, err := renderer.RenderSelection(selection)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify command
	expectedCmd := "opencode --model claude-sonnet --agent coder"
	if spec.RenderedCommand != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, spec.RenderedCommand)
	}

	// Verify prompt
	expectedPrompt := "bb-test: Test Ticket\nA test description"
	if spec.RenderedPrompt != expectedPrompt {
		t.Errorf("Expected prompt %q, got %q", expectedPrompt, spec.RenderedPrompt)
	}

	// Verify window name is ticket ID
	if spec.WindowName != "bb-test" {
		t.Errorf("Expected window name 'bb-test', got %q", spec.WindowName)
	}

	// Verify selection is preserved
	if spec.Selection.Ticket.ID != "bb-test" {
		t.Errorf("Expected ticket ID preserved")
	}
}

func TestRenderer_RenderSelection_NoPrompt(t *testing.T) {
	renderer := NewRenderer()

	selection := domain.Selection{
		Ticket: domain.Ticket{
			ID:    "bb-123",
			Title: "No Prompt Test",
		},
		Harness: domain.Harness{
			Name:            "minimal",
			CommandTemplate: "minimal",
			PromptTemplate:  "", // No prompt
		},
		Model: "test-model",
		Agent: "test-agent",
	}

	spec, err := renderer.RenderSelection(selection)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if spec.RenderedPrompt != "" {
		t.Errorf("Expected empty prompt, got %q", spec.RenderedPrompt)
	}

	if spec.RenderedCommand != "minimal" {
		t.Errorf("Expected command 'minimal', got %q", spec.RenderedCommand)
	}
}

func TestRenderer_RenderSelection_CommandError(t *testing.T) {
	renderer := NewRenderer()

	selection := domain.Selection{
		Ticket: domain.Ticket{
			ID: "bb-123",
		},
		Harness: domain.Harness{
			Name:            "bad-cmd",
			CommandTemplate: "{{.UndefinedVar}}", // Will fail on execution
		},
	}

	_, err := renderer.RenderSelection(selection)
	if err == nil {
		t.Fatal("Expected error for invalid command template")
	}

	if !strings.Contains(err.Error(), "failed to render command") {
		t.Errorf("Error should mention command rendering, got: %v", err)
	}
}

func TestRenderer_RenderSelection_PromptError(t *testing.T) {
	renderer := NewRenderer()

	selection := domain.Selection{
		Ticket: domain.Ticket{
			ID: "bb-123",
		},
		Harness: domain.Harness{
			Name:            "bad-prompt",
			CommandTemplate: "echo ok",
			PromptTemplate:  "{{.UndefinedVar}}", // Will fail on execution
		},
	}

	_, err := renderer.RenderSelection(selection)
	if err == nil {
		t.Fatal("Expected error for invalid prompt template")
	}

	if !strings.Contains(err.Error(), "failed to render prompt") {
		t.Errorf("Error should mention prompt rendering, got: %v", err)
	}
}

func TestBuildTemplateContext_Complete(t *testing.T) {
	now := time.Now()
	selection := domain.Selection{
		Ticket: domain.Ticket{
			ID:          "bb-abc",
			Title:       "Test Title",
			Description: "Test Desc",
			Status:      "in_progress",
			Priority:    2,
			IssueType:   "feature",
			Assignee:    "user1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Harness: domain.Harness{
			Name: "opencode",
		},
		Model: "claude-sonnet",
		Agent: "coder",
	}

	ctx := BuildTemplateContext(selection)

	// Verify ticket fields
	if ctx.TicketID != "bb-abc" {
		t.Errorf("Expected TicketID 'bb-abc', got %q", ctx.TicketID)
	}
	if ctx.TicketTitle != "Test Title" {
		t.Errorf("Expected TicketTitle 'Test Title', got %q", ctx.TicketTitle)
	}
	if ctx.TicketDescription != "Test Desc" {
		t.Errorf("Expected TicketDescription 'Test Desc', got %q", ctx.TicketDescription)
	}
	if ctx.TicketStatus != "in_progress" {
		t.Errorf("Expected TicketStatus 'in_progress', got %q", ctx.TicketStatus)
	}
	if ctx.TicketPriority != 2 {
		t.Errorf("Expected TicketPriority 2, got %d", ctx.TicketPriority)
	}
	if ctx.TicketIssueType != "feature" {
		t.Errorf("Expected TicketIssueType 'feature', got %q", ctx.TicketIssueType)
	}
	if ctx.TicketAssignee != "user1" {
		t.Errorf("Expected TicketAssignee 'user1', got %q", ctx.TicketAssignee)
	}

	// Verify harness fields
	if ctx.HarnessName != "opencode" {
		t.Errorf("Expected HarnessName 'opencode', got %q", ctx.HarnessName)
	}

	// Verify selection fields
	if ctx.Model != "claude-sonnet" {
		t.Errorf("Expected Model 'claude-sonnet', got %q", ctx.Model)
	}
	if ctx.Agent != "coder" {
		t.Errorf("Expected Agent 'coder', got %q", ctx.Agent)
	}

	// Verify timestamp uses UpdatedAt
	if !ctx.Timestamp.Equal(now) {
		t.Error("Expected Timestamp to equal Ticket UpdatedAt")
	}
}

func TestRenderer_MultilineTemplate(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name: "multiline",
		PromptTemplate: `Ticket: {{.TicketID}}
Title: {{.TicketTitle}}
Priority: {{.TicketPriority}}`,
	}
	ctx := domain.TemplateContext{
		TicketID:       "bb-456",
		TicketTitle:    "Multiline Test",
		TicketPriority: 1,
	}

	result, err := renderer.RenderPrompt(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "Ticket: bb-456\nTitle: Multiline Test\nPriority: 1"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderer_Escaping(t *testing.T) {
	renderer := NewRenderer()
	harness := domain.Harness{
		Name:            "escape",
		CommandTemplate: "echo '{{.TicketTitle}}'",
	}
	ctx := domain.TemplateContext{
		TicketTitle: `It's a "test"`, // Contains quotes
	}

	result, err := renderer.RenderCommand(harness, ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// text/template does NOT escape by default (unlike html/template)
	expected := `echo 'It's a "test"'`
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

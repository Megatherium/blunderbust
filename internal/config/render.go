// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/megatherium/blunderbust/internal/domain"
)

// Renderer handles template rendering for harness configurations.
type Renderer struct{}

// NewRenderer creates a new template renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// RenderCommand renders the command template for a harness with the given context.
// Returns the rendered command string or an error with context about which harness failed.
func (r *Renderer) RenderCommand(harness domain.Harness, ctx domain.TemplateContext) (string, error) {
	return r.renderTemplate(
		harness.Name,
		"command_template",
		harness.CommandTemplate,
		ctx,
	)
}

// RenderPrompt renders the prompt template for a harness with the given context.
// If the harness has no prompt_template, returns an empty string with no error.
// Returns the rendered prompt string or an error with context about which harness failed.
func (r *Renderer) RenderPrompt(harness domain.Harness, ctx domain.TemplateContext) (string, error) {
	if harness.PromptTemplate == "" {
		return "", nil
	}
	return r.renderTemplate(
		harness.Name,
		"prompt_template",
		harness.PromptTemplate,
		ctx,
	)
}

// renderTemplate executes a Go text/template with the given context.
func (r *Renderer) renderTemplate(harnessName, templateName, templateStr string, ctx domain.TemplateContext) (string, error) {
	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf(
			"failed to parse %s for harness %q: %w",
			templateName,
			harnessName,
			err,
		)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf(
			"failed to execute %s for harness %q: %w",
			templateName,
			harnessName,
			err,
		)
	}

	return buf.String(), nil
}

// RenderSelection renders both command and prompt for a complete selection.
// Returns a LaunchSpec with all fields populated.
// Note: Prompt is rendered before command to allow {{.Prompt}} in command templates.
func (r *Renderer) RenderSelection(selection domain.Selection) (*domain.LaunchSpec, error) {
	ctx := BuildTemplateContext(selection)

	// Render prompt first so it's available in the context for command rendering
	renderedPrompt, err := r.RenderPrompt(selection.Harness, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render prompt: %w", err)
	}

	// Update context with rendered prompt for use in command template
	ctx.Prompt = renderedPrompt

	// Now render command with the updated context (which includes Prompt)
	renderedCmd, err := r.RenderCommand(selection.Harness, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render command: %w", err)
	}

	return &domain.LaunchSpec{
		Selection:       selection,
		RenderedCommand: renderedCmd,
		RenderedPrompt:  renderedPrompt,
		WindowName:      selection.Ticket.ID,
	}, nil
}

// BuildTemplateContext creates a TemplateContext from a Selection.
// This is the single source of truth for mapping Selection to TemplateContext.
func BuildTemplateContext(sel domain.Selection) domain.TemplateContext {
	return domain.TemplateContext{
		// Ticket fields
		TicketID:          sel.Ticket.ID,
		TicketTitle:       sel.Ticket.Title,
		TicketDescription: sel.Ticket.Description,
		TicketStatus:      sel.Ticket.Status,
		TicketPriority:    sel.Ticket.Priority,
		TicketIssueType:   sel.Ticket.IssueType,
		TicketAssignee:    sel.Ticket.Assignee,
		TicketCreatedAt:   sel.Ticket.CreatedAt,
		TicketUpdatedAt:   sel.Ticket.UpdatedAt,

		// Harness fields
		HarnessName: sel.Harness.Name,

		// Selection fields
		Model: sel.Model,
		Agent: sel.Agent,

		// Environment fields (populated by caller if needed)
		RepoPath: "",
		Branch:   "",
		WorkDir:  "",
		User:     "",
		Hostname: "",

		// Runtime fields (populated by caller if needed)
		DryRun:    false,
		Debug:     false,
		Timestamp: sel.Ticket.UpdatedAt,

		// Prompt field - will be populated during RenderSelection
		// if a prompt_template is configured
		Prompt: "",
	}
}

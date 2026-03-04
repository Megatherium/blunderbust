// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package domain

import (
	"strings"
	"time"
)

// ModelContext keeps the full model ID while exposing structured accessors in templates.
// Using a string-backed type preserves template truthiness semantics for {{if .Model}}.
type ModelContext string

// NewModelContext wraps a model ID for template access.
func NewModelContext(modelID string) ModelContext { return ModelContext(modelID) }

// String preserves backward compatibility with templates that use {{.Model}}.
func (m ModelContext) String() string {
	return string(m)
}

// ModelID returns the full model identifier.
func (m ModelContext) ModelID() string {
	return string(m)
}

// Provider returns the provider segment, if present.
func (m ModelContext) Provider() string {
	provider, _, _ := m.parts()
	return provider
}

// Org returns the organization segment, if present.
func (m ModelContext) Org() string {
	_, org, _ := m.parts()
	return org
}

// Organization aliases Org for template readability.
func (m ModelContext) Organization() string {
	return m.Org()
}

// Name returns the model name segment.
func (m ModelContext) Name() string {
	_, _, name := m.parts()
	return name
}

func (m ModelContext) parts() (provider, org, name string) {
	if m == "" {
		return "", "", ""
	}

	parts := strings.Split(string(m), "/")
	switch len(parts) {
	case 1:
		return "", "", parts[0]
	case 2:
		return parts[0], "", parts[1]
	default:
		return parts[0], parts[1], strings.Join(parts[2:], "/")
	}
}

// TemplateContext is the fat context passed to both command and prompt
// templates. It is intentionally generous — templates pick what they need.
type TemplateContext struct {
	// Ticket fields
	TicketID          string
	TicketTitle       string
	TicketDescription string
	TicketStatus      string
	TicketPriority    int
	TicketIssueType   string
	TicketAssignee    string
	TicketCreatedAt   time.Time
	TicketUpdatedAt   time.Time

	// Harness fields
	HarnessName string

	// Selection fields
	Model ModelContext
	Agent string

	// Environment fields
	RepoPath string
	Branch   string
	WorkDir  string
	User     string
	Hostname string

	// Runtime fields
	DryRun    bool
	Debug     bool
	Timestamp time.Time

	// Prompt field for command template access
	// This is populated with the rendered prompt text (from prompt_template)
	// and can be referenced in command_template using {{.Prompt}}
	// If no prompt_template is configured, this field will be empty.
	Prompt string
}

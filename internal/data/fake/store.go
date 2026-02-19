// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package fake

import (
	"context"
	"strings"
	"time"

	"github.com/megatherium/blunderbuss/internal/data"
	"github.com/megatherium/blunderbuss/internal/domain"
)

// TicketStore is an in-memory fake implementing data.TicketStore.
type TicketStore struct {
	Tickets []domain.Ticket
}

// Verify interface compliance at compile time.
var _ data.TicketStore = (*TicketStore)(nil)

// ListTickets returns tickets matching the given filter.
func (s *TicketStore) ListTickets(_ context.Context, filter data.TicketFilter) ([]domain.Ticket, error) {
	var results []domain.Ticket
	for i := range s.Tickets {
		t := &s.Tickets[i]
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.IssueType != "" && t.IssueType != filter.IssueType {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(t.Title), strings.ToLower(filter.Search)) {
			continue
		}
		results = append(results, *t)
		if filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	return results, nil
}

// NewWithSampleData returns a FakeTicketStore pre-loaded with sample tickets.
func NewWithSampleData() *TicketStore {
	now := time.Now()
	return &TicketStore{
		Tickets: []domain.Ticket{
			{ID: "bb-001", Title: "Bootstrap Go module", Status: "closed", Priority: 1, IssueType: "task", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour)},
			{ID: "bb-002", Title: "Define core domain types", Status: "open", Priority: 1, IssueType: "task", CreatedAt: now.Add(-24 * time.Hour), UpdatedAt: now},
			{ID: "bb-003", Title: "Implement TicketStore backend", Status: "open", Priority: 1, IssueType: "task", CreatedAt: now.Add(-12 * time.Hour), UpdatedAt: now},
			{ID: "bb-004", Title: "Build TUI skeleton", Status: "open", Priority: 1, IssueType: "feature", CreatedAt: now.Add(-6 * time.Hour), UpdatedAt: now},
			{ID: "bb-005", Title: "Implement tmux launcher", Status: "open", Priority: 2, IssueType: "task", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now},
		},
	}
}

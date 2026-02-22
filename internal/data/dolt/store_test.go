// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dolt

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/megatherium/blunderbust/internal/data"
)

func TestStore_ListTickets_NoFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 ORDER BY priority ASC, updated_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-001", "Test Ticket", "Description", "open", 1, "task", nil, time.Now(), time.Now()).
			AddRow("bb-002", "Another Ticket", "Another desc", "open", 2, "feature", "user@example.com", time.Now(), time.Now()))

	filter := data.TicketFilter{}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}

	if tickets[0].ID != "bb-001" {
		t.Errorf("expected first ticket ID 'bb-001', got %q", tickets[0].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_WithStatusFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND status = \? ORDER BY priority ASC, updated_at DESC`).
		WithArgs("closed").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-003", "Closed Ticket", "Done", "closed", 1, "task", nil, time.Now(), time.Now()))

	filter := data.TicketFilter{Status: "closed"}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 1 {
		t.Errorf("expected 1 ticket, got %d", len(tickets))
	}

	if tickets[0].Status != "closed" {
		t.Errorf("expected status 'closed', got %q", tickets[0].Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_WithIssueTypeFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND issue_type = \? ORDER BY priority ASC, updated_at DESC`).
		WithArgs("feature").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-004", "Feature Ticket", "New feature", "open", 1, "feature", nil, time.Now(), time.Now()))

	filter := data.TicketFilter{IssueType: "feature"}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 1 {
		t.Errorf("expected 1 ticket, got %d", len(tickets))
	}

	if tickets[0].IssueType != "feature" {
		t.Errorf("expected issue_type 'feature', got %q", tickets[0].IssueType)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_WithSearchFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND title LIKE \? ORDER BY priority ASC, updated_at DESC`).
		WithArgs("%test%").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-005", "Test Ticket", "A test", "open", 1, "task", nil, time.Now(), time.Now()).
			AddRow("bb-006", "Testing Again", "Another test", "open", 2, "task", nil, time.Now(), time.Now()))

	filter := data.TicketFilter{Search: "test"}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_WithLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 ORDER BY priority ASC, updated_at DESC LIMIT \?`).
		WithArgs(5).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-007", "Ticket 1", "Desc 1", "open", 1, "task", nil, time.Now(), time.Now()).
			AddRow("bb-008", "Ticket 2", "Desc 2", "open", 2, "task", nil, time.Now(), time.Now()))

	filter := data.TicketFilter{Limit: 5}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_CombinedFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND status = \? AND issue_type = \? AND title LIKE \? ORDER BY priority ASC, updated_at DESC LIMIT \?`).
		WithArgs("open", "bug", "%crash%", 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-009", "Crash bug", "It crashes", "open", 0, "bug", nil, time.Now(), time.Now()))

	filter := data.TicketFilter{
		Status:    "open",
		IssueType: "bug",
		Search:    "crash",
		Limit:     10,
	}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 1 {
		t.Errorf("expected 1 ticket, got %d", len(tickets))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_WithAssignee(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	assignee := "user@example.com"
	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 ORDER BY priority ASC, updated_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-010", "Assigned Ticket", "Work", "open", 1, "task", assignee, time.Now(), time.Now()))

	filter := data.TicketFilter{}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}

	if tickets[0].Assignee != assignee {
		t.Errorf("expected assignee %q, got %q", assignee, tickets[0].Assignee)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_EmptyResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	mock.ExpectQuery(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 ORDER BY priority ASC, updated_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}))

	filter := data.TicketFilter{}
	tickets, err := store.ListTickets(context.Background(), filter)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListTickets_StoreClosed(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db, mode: EmbeddedMode, closed: true}

	filter := data.TicketFilter{}
	_, err = store.ListTickets(context.Background(), filter)

	if err == nil {
		t.Fatal("expected error for closed store")
	}

	if err.Error() != "store is closed" {
		t.Errorf("expected 'store is closed' error, got: %v", err)
	}
}

func TestBuildListTicketsQuery(t *testing.T) {
	tests := []struct {
		name     string
		filter   data.TicketFilter
		expected string
		args     []any
	}{
		{
			name:     "no filters",
			filter:   data.TicketFilter{},
			expected: "SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 ORDER BY priority ASC, updated_at DESC",
			args:     nil,
		},
		{
			name:     "status filter",
			filter:   data.TicketFilter{Status: "open"},
			expected: "SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND status = ? ORDER BY priority ASC, updated_at DESC",
			args:     []any{"open"},
		},
		{
			name:     "issue type filter",
			filter:   data.TicketFilter{IssueType: "bug"},
			expected: "SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND issue_type = ? ORDER BY priority ASC, updated_at DESC",
			args:     []any{"bug"},
		},
		{
			name:     "search filter",
			filter:   data.TicketFilter{Search: "test"},
			expected: "SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND title LIKE ? ORDER BY priority ASC, updated_at DESC",
			args:     []any{"%test%"},
		},
		{
			name:     "limit filter",
			filter:   data.TicketFilter{Limit: 10},
			expected: "SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 ORDER BY priority ASC, updated_at DESC LIMIT ?",
			args:     []any{10},
		},
		{
			name:     "combined filters",
			filter:   data.TicketFilter{Status: "open", IssueType: "feature", Search: "auth", Limit: 5},
			expected: "SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1 AND status = ? AND issue_type = ? AND title LIKE ? ORDER BY priority ASC, updated_at DESC LIMIT ?",
			args:     []any{"open", "feature", "%auth%", 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildListTicketsQuery(tt.filter)

			if query != tt.expected {
				t.Errorf("query mismatch\nexpected: %s\ngot:      %s", tt.expected, query)
			}

			if len(args) != len(tt.args) {
				t.Errorf("args length mismatch: expected %d, got %d", len(tt.args), len(args))
			}

			for i := range tt.args {
				if args[i] != tt.args[i] {
					t.Errorf("arg[%d] mismatch: expected %v, got %v", i, tt.args[i], args[i])
				}
			}
		})
	}
}

func TestScanTickets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()

	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "status", "priority", "issue_type", "assignee", "created_at", "updated_at",
		}).
			AddRow("bb-001", "Test", "Description", "open", 1, "task", nil, now, now).
			AddRow("bb-002", "Assigned", "Work", "in_progress", 2, "feature", "dev@example.com", now, now))

	rows, err := db.Query(`SELECT`)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	tickets, err := scanTickets(rows)
	if err != nil {
		t.Fatalf("failed to scan tickets: %v", err)
	}

	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets, got %d", len(tickets))
	}

	// First ticket has no assignee
	if tickets[0].Assignee != "" {
		t.Errorf("expected empty assignee for first ticket, got %q", tickets[0].Assignee)
	}

	// Second ticket has assignee
	if tickets[1].Assignee != "dev@example.com" {
		t.Errorf("expected assignee 'dev@example.com', got %q", tickets[1].Assignee)
	}

	if tickets[0].ID != "bb-001" || tickets[1].ID != "bb-002" {
		t.Error("ticket IDs don't match expected")
	}
}

func TestVerifySchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Schema verification passes
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM ready_issues LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	if err := verifySchema(context.Background(), db); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestVerifySchema_Failure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	// Schema verification fails
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM ready_issues LIMIT 1`).
		WillReturnError(sqlmock.ErrCancelled)

	err = verifySchema(context.Background(), db)
	if err == nil {
		t.Fatal("expected error for schema verification failure")
	}

	if !contains(err.Error(), "schema verification failed") {
		t.Errorf("error should mention schema verification, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestBuildServerDSN(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		expected string
	}{
		{
			name: "full config",
			metadata: Metadata{
				DoltDatabase: "beads_test",
				ServerHost:   "10.11.0.1",
				ServerPort:   13307,
				ServerUser:   "mysql-root",
			},
			expected: "mysql-root@tcp(10.11.0.1:13307)/beads_test?parseTime=true",
		},
		{
			name: "with password",
			metadata: Metadata{
				DoltDatabase: "beads_prod",
				ServerHost:   "db.example.com",
				ServerPort:   3306,
				ServerUser:   "admin",
			},
			expected: "admin:secret123@tcp(db.example.com:3306)/beads_prod?parseTime=true",
		},
		{
			name: "defaults for missing fields",
			metadata: Metadata{
				DoltDatabase: "beads_default",
			},
			expected: "root@tcp(127.0.0.1:3306)/beads_default?parseTime=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set password env var for the with_password test
			if tt.name == "with password" {
				t.Setenv("BEADS_DOLT_PASSWORD", "secret123")
			}

			got := buildServerDSN(&tt.metadata)
			if got != tt.expected {
				t.Errorf("DSN mismatch\nexpected: %s\ngot:      %s", tt.expected, got)
			}
		})
	}
}

func TestStore_Close_Idempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	// Note: we don't defer db.Close() here because we're testing store.Close()

	store := &Store{db: db, mode: EmbeddedMode}

	// First close should succeed
	mock.ExpectClose()
	if err := store.Close(); err != nil {
		t.Errorf("first Close() failed: %v", err)
	}

	// Second close should succeed without error (idempotent)
	if err := store.Close(); err != nil {
		t.Errorf("second Close() should be idempotent but failed: %v", err)
	}

	// Verify store is marked as closed
	if !store.closed {
		t.Error("store.closed should be true after Close()")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestStore_Close_ReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	store := &Store{db: db, mode: EmbeddedMode}

	// Simulate error on db.Close()
	mock.ExpectClose().WillReturnError(fmt.Errorf("connection reset"))

	err = store.Close()
	if err == nil {
		t.Fatal("expected error from Close()")
	}

	if err.Error() != "connection reset" {
		t.Errorf("expected 'connection reset' error, got: %v", err)
	}

	// Store should still be marked as closed even on error
	if !store.closed {
		t.Error("store.closed should be true even when Close() returns error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

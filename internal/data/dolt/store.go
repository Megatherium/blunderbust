// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/domain"
)

// Store implements data.TicketStore using Dolt.
type Store struct {
	db     *sql.DB
	mode   Mode
	closed bool
}

// Verify interface compliance at compile time.
var _ data.TicketStore = (*Store)(nil)

// NewStore creates a TicketStore connected to a Dolt database.
// It reads metadata.json from the beads directory to determine connection mode.
//
// For embedded mode: opens .beads/dolt/ using the embedded driver.
// For server mode: connects to the configured dolt sql-server.
func NewStore(ctx context.Context, opts domain.AppOptions) (*Store, error) {
	beadsDir := opts.BeadsDir
	if beadsDir == "" {
		beadsDir = ".beads"
	}

	metadata, err := LoadMetadata(beadsDir)
	if err != nil {
		return nil, err
	}

	switch metadata.ConnectionMode() {
	case ServerMode:
		if opts.Debug {
			fmt.Fprintf(os.Stderr, "Dolt server mode enabled\n")
		}
		return newServerStore(ctx, beadsDir, metadata)
	case EmbeddedMode:
		if opts.Debug {
			fmt.Fprintf(os.Stderr, "Dolt embedded mode enabled\n")
		}
		return newEmbeddedStore(ctx, beadsDir, metadata)
	default:
		return nil, fmt.Errorf("unknown connection mode: %v", metadata.ConnectionMode())
	}
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.db.Close()
}

// ListTickets queries the ready_issues view and returns tickets matching the filter.
func (s *Store) ListTickets(ctx context.Context, filter data.TicketFilter) ([]domain.Ticket, error) {
	if s.closed {
		return nil, fmt.Errorf("store is closed")
	}

	query, args := buildListTicketsQuery(filter)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}
	defer rows.Close()

	return scanTickets(rows)
}

// buildListTicketsQuery constructs the SQL query with optional filters.
func buildListTicketsQuery(filter data.TicketFilter) (query string, args []any) {
	// Base query - we select specific fields from ready_issues view
	// ready_issues already filters for unblocked, non-deferred, non-ephemeral issues
	var sb strings.Builder
	sb.WriteString(`SELECT id, title, description, status, priority, issue_type, assignee, created_at, updated_at FROM ready_issues WHERE 1=1`)

	if filter.Status != "" {
		sb.WriteString(" AND status = ?")
		args = append(args, filter.Status)
	}

	if filter.IssueType != "" {
		sb.WriteString(" AND issue_type = ?")
		args = append(args, filter.IssueType)
	}

	if filter.Search != "" {
		sb.WriteString(" AND title LIKE ?")
		args = append(args, "%"+filter.Search+"%")
	}

	// Order by priority (lower number = higher priority), then by updated_at (most recent first)
	sb.WriteString(" ORDER BY priority ASC, updated_at DESC")

	if filter.Limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, filter.Limit)
	}

	return sb.String(), args
}

// scanTickets reads rows from the result set and converts them to domain.Ticket.
func scanTickets(rows *sql.Rows) ([]domain.Ticket, error) {
	var tickets []domain.Ticket

	for rows.Next() {
		var t domain.Ticket
		var assignee sql.NullString

		err := rows.Scan(
			&t.ID,
			&t.Title,
			&t.Description,
			&t.Status,
			&t.Priority,
			&t.IssueType,
			&assignee,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket row: %w", err)
		}

		if assignee.Valid {
			t.Assignee = assignee.String
		}

		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ticket rows: %w", err)
	}

	return tickets, nil
}

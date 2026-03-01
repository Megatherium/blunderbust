// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dolt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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

// ErrServerNotRunning is returned when the Dolt server is not running and autostart is disabled.
type ErrServerNotRunning struct {
	Message string
}

func (e *ErrServerNotRunning) Error() string {
	return e.Message
}

// IsErrServerNotRunning returns true if the error is an ErrServerNotRunning.
func IsErrServerNotRunning(err error) bool {
	var e *ErrServerNotRunning
	return errors.As(err, &e)
}

// NewStore creates a TicketStore connected to a Dolt database.
// It reads metadata.json from the beads directory to determine connection mode.
//
// For embedded mode: opens .beads/dolt/ using the embedded driver.
// For server mode: connects to the configured dolt sql-server.
// If autostart is true and the server is not running, it will attempt to start it.
func NewStore(ctx context.Context, opts domain.AppOptions, autostart bool) (*Store, error) {
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
		// Resolve server port if not explicitly configured
		if metadata.ServerPort == 0 {
			resolvedPort, err := metadata.ResolveServerPort(beadsDir)
			if err != nil && opts.Debug {
				fmt.Fprintf(os.Stderr, "Warning: failed to auto-detect Dolt port: %v\n", err)
			}
			if resolvedPort > 0 && opts.Debug {
				fmt.Fprintf(os.Stderr, "Auto-detected Dolt server port: %d\n", resolvedPort)
			}
		}
		store, err := newServerStore(ctx, beadsDir, metadata)
		if err != nil {
			// Check if it's a connection error
			if isConnectionError(err) {
				if autostart {
					if opts.Debug {
						fmt.Fprintf(os.Stderr, "Dolt server not running, attempting to start...\n")
					}
					if startErr := StartServer(beadsDir, 30*time.Second); startErr != nil {
						return nil, fmt.Errorf("failed to auto-start dolt server: %w", startErr)
					}
					// Retry connection after starting server
					return newServerStore(ctx, beadsDir, metadata)
				}
				// Autostart is disabled, return special error
				return nil, &ErrServerNotRunning{
					Message: "Dolt server is not running. Start dolt server? [y/N]",
				}
			}
			return nil, err
		}
		return store, nil
	case EmbeddedMode:
		if opts.Debug {
			fmt.Fprintf(os.Stderr, "Dolt embedded mode enabled\n")
		}
		return newEmbeddedStore(ctx, beadsDir, metadata)
	default:
		return nil, fmt.Errorf("unknown connection mode: %v", metadata.ConnectionMode())
	}
}

// isConnectionError returns true if the error indicates the server is not running.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "cannot connect to Dolt server") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "No connection could be made") ||
		strings.Contains(errStr, "dial tcp")
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.db.Close()
}

// DB exposes the underlying SQL connection for advanced queries
func (s *Store) DB() *sql.DB {
	return s.db
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

// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package dolt implements the TicketStore interface using Dolt databases.
//
// Dolt is a MySQL-compatible database with Git-like version control.
// This package supports two connection modes:
//
// # Embedded Mode
//
// Uses github.com/dolthub/driver (requires CGO). The database is stored
// locally in .beads/dolt/ and supports only a single concurrent connection.
//
// NOTE: Embedded mode is only available when building with the "embedded" build tag:
//
//	go build -tags=embedded ./cmd/blunderbust
//	make build-full
//
// Default builds (no build tags) do not include embedded support and will return
// an error if embedded mode is requested. This reduces binary size from ~93MB to ~20-30MB.
//
// DSN format: file://<abs_path>?database=<name>&commitname=<name>&commitemail=<email>
//
// # Server Mode
//
// Connects to a running dolt sql-server via the MySQL protocol using
// github.com/go-sql-driver/mysql. Supports multiple concurrent connections.
// Activated by setting dolt_mode: server or providing server connection
// details in metadata.json.
//
// Server mode is available in all builds (default and with -tags=embedded).
//
// DSN format: user:password@tcp(host:port)/database?parseTime=true&loc=UTC
//
// Usage
//
//	store, err := dolt.NewStore(ctx, domain.AppOptions{BeadsDir: ".beads"})
//	if err != nil {
//		return err
//	}
//	defer store.Close()
//
//	tickets, err := store.ListTickets(ctx, data.TicketFilter{
//	    Status: "open",
//	    Limit:  10,
//	})
//
// # Error Handling
//
// All errors include actionable context. Common errors:
//
//   - "no beads database found": metadata.json is missing
//   - "dolt database directory not found": .beads/dolt/ directory is missing
//   - "failed to connect": Database is corrupted or locked
//   - "schema verification failed": Missing or incompatible schema
//
// Both modes query the ready_issues view which filters for unblocked,
// non-deferred, non-ephemeral issues.
//
// Note: This package provides read-only access. Blunderbust never writes
// to the beads database.
package dolt

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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// newServerStore creates a Store connected to a Dolt sql-server.
// Uses standard MySQL driver for remote connections.
// Note: beadsDir parameter is unused in server mode since we connect to a
// remote server rather than a local database directory.
func newServerStore(ctx context.Context, _ string, metadata *Metadata) (*Store, error) {
	// Build MySQL DSN
	dsn := buildServerDSN(metadata)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create MySQL connection pool: %w", err)
	}

	// Configure connection pool for server mode
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf(
			"cannot connect to Dolt server at %s:%d: %w; "+
				"check that the server is running and accessible",
			metadata.ServerHost, metadata.ServerPort, err,
		)
	}

	// Verify the database schema is accessible
	if err := verifySchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		db:   db,
		mode: ServerMode,
	}, nil
}

// buildServerDSN constructs the MySQL DSN from metadata using mysql.Config
// for proper escaping of special characters in passwords.
func buildServerDSN(metadata *Metadata) string {
	cfg := mysql.NewConfig()

	// Set defaults
	cfg.Net = "tcp"
	cfg.User = metadata.ServerUser
	if cfg.User == "" {
		cfg.User = "root"
	}

	// Build address with defaults
	host := metadata.ServerHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := metadata.ServerPort
	if port == 0 {
		port = 3307 // Default Dolt sql-server port
	}
	cfg.Addr = fmt.Sprintf("%s:%d", host, port)

	// Check for password in environment
	cfg.Passwd = os.Getenv("BEADS_DOLT_PASSWORD")

	cfg.DBName = metadata.DoltDatabase

	// Connection parameters
	cfg.ParseTime = true
	cfg.Loc = time.UTC

	return cfg.FormatDSN()
}

// StartServer attempts to start the Dolt server by running 'bd dolt start'.
// It waits for the server to be ready by polling with exponential backoff.
// Returns an error if the server fails to start or doesn't become ready within the timeout.
func StartServer(beadsDir string, timeout time.Duration) error {
	projectDir := filepath.Dir(beadsDir)

	// Run 'bd dolt start' with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bd", "dolt", "start")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dolt server: %w", err)
	}

	// Wait for the command to complete (it should daemonize)
	if err := cmd.Wait(); err != nil {
		// Check if it's just the context timeout (server still starting)
		if ctx.Err() == context.DeadlineExceeded {
			// This is expected if the server takes time to start
			// Continue to polling below
		} else {
			return fmt.Errorf("dolt server start command failed: %w", err)
		}
	}

	// Poll for server readiness
	return waitForServerReady(beadsDir, 10*time.Second)
}

// waitForServerReady polls the Dolt server status until it's ready or timeout.
func waitForServerReady(beadsDir string, timeout time.Duration) error {
	projectDir := filepath.Dir(beadsDir)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.Command("bd", "dolt", "status")
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()

		if err == nil {
			outputStr := string(output)
			// Check if server is running
			if strings.Contains(outputStr, "Status: running") ||
				strings.Contains(outputStr, "Port:") {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("dolt server did not become ready within %v", timeout)
}

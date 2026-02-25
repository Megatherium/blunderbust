// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dolt

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
)

// Mode represents the Dolt connection mode.
type Mode string

const (
	// EmbeddedMode uses github.com/dolthub/driver (CGO required).
	// Database is stored locally in .beads/dolt/
	EmbeddedMode Mode = "embedded"
	// ServerMode connects to a running dolt sql-server via MySQL protocol.
	ServerMode Mode = "server"
)

// Metadata represents the parsed .beads/metadata.json file.
type Metadata struct {
	// Database backend type (should be "dolt")
	Backend string `json:"backend"`
	// DoltDatabase is the database name within Dolt (e.g., "beads_bb")
	DoltDatabase string `json:"dolt_database"`
	// DoltMode indicates whether to use embedded or server mode
	DoltMode string `json:"dolt_mode"`
	// ServerHost is the hostname for server mode connections
	ServerHost string `json:"dolt_server_host"`
	// ServerPort is the port for server mode connections
	ServerPort int `json:"dolt_server_port"`
	// ServerUser is the MySQL user for server mode connections
	ServerUser string `json:"dolt_server_user"`
}

// ConnectionMode determines the connection mode from the metadata.
// Returns ServerMode if dolt_mode is "server" or if server connection
// fields are present. Otherwise returns EmbeddedMode.
func (m *Metadata) ConnectionMode() Mode {
	if m.DoltMode == "server" {
		return ServerMode
	}
	// Also detect server mode by presence of server fields
	if m.ServerHost != "" && m.ServerPort > 0 {
		return ServerMode
	}
	return EmbeddedMode
}

// IsValid returns true if the metadata contains the minimum required fields.
func (m *Metadata) IsValid() bool {
	return m.DoltDatabase != ""
}

// LoadMetadata reads and parses the metadata.json file from the given beads directory.
// Returns actionable errors for common failure scenarios.
func LoadMetadata(beadsDir string) (*Metadata, error) {
	metadataPath := filepath.Join(beadsDir, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"no beads database found at %q: metadata.json is missing\n"+
					"Is this a beads project? Run 'bd init' to initialize beads in this repository",
				beadsDir,
			)
		}
		return nil, fmt.Errorf("failed to read metadata.json: %w", err)
	}

	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf(
			"metadata.json is corrupted or has invalid JSON: %w\n"+
				"Try removing %s and running 'bd init' to recreate it",
			err, metadataPath,
		)
	}

	if !metadata.IsValid() {
		return nil, fmt.Errorf(
			"metadata.json is missing required field 'dolt_database'\n"+
				"File location: %s\n"+
				"Try running 'bd init' to regenerate the metadata file",
			metadataPath,
		)
	}

	return &metadata, nil
}

// ResolveServerPort attempts to detect the Dolt server port if not explicitly configured.
// It first checks if ServerPort is already set in metadata. If not, it runs 'bd dolt status'
// to extract the expected port from the output. Returns the resolved port and any error
// encountered during detection.
func (m *Metadata) ResolveServerPort(beadsDir string) (int, error) {
	// If port is already configured, use it
	if m.ServerPort > 0 {
		return m.ServerPort, nil
	}

	// Try to detect port from 'bd dolt status'
	port, err := m.detectPortFromDoltStatus(beadsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to detect Dolt server port: %w", err)
	}

	if port > 0 {
		m.ServerPort = port
		return port, nil
	}

	// Return 0 to indicate no port detected - caller should use default
	return 0, nil
}

// detectPortFromDoltStatus runs 'bd dolt status' and extracts the port from output.
// It handles both running server ("Port: 1234") and stopped server ("Expected port: 1234") outputs.
func (m *Metadata) detectPortFromDoltStatus(beadsDir string) (int, error) {
	// Get the parent directory of beadsDir (project root)
	projectDir := filepath.Dir(beadsDir)

	cmd := exec.Command("bd", "dolt", "status")
	cmd.Dir = projectDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		// bd command might fail if dolt is not initialized, which is okay
		// we'll fall back to default port
		return 0, nil
	}

	outputStr := string(output)

	// Look for port patterns in order of specificity:
	// 1. "Port: XXXX" (when server is running)
	// 2. "Expected port: XXXX" (when server is not running)
	patterns := []string{
		`(?:^|\s)Port:\s*(\d+)`,
		`Expected port:\s*(\d+)`,
	}

	for _, pattern := range patterns {
		portRegex := regexp.MustCompile(pattern)
		matches := portRegex.FindStringSubmatch(outputStr)

		if len(matches) >= 2 {
			port, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, fmt.Errorf("failed to parse port number from 'bd dolt status': %w", err)
			}
			// Validate port range (typical Dolt ports are 1024-65535)
			if port >= 1024 && port <= 65535 {
				return port, nil
			}
		}
	}

	// No port found in output - return 0 to indicate detection failed
	return 0, nil
}

// DoltDir returns the path to the Dolt database directory.
// This is always beadsDir/dolt for embedded mode.
func DoltDir(beadsDir string) string {
	return filepath.Join(beadsDir, "dolt")
}

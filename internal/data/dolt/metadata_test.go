// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package dolt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMetadata_EmbeddedMode(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	metadataJSON := `{
		"database": "dolt",
		"backend": "dolt",
		"dolt_database": "beads_bb"
	}`
	metadataPath := filepath.Join(beadsDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(metadataJSON), 0644); err != nil {
		t.Fatalf("Failed to write metadata.json: %v", err)
	}

	metadata, err := LoadMetadata(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.DoltDatabase != "beads_bb" {
		t.Errorf("Expected DoltDatabase='beads_bb', got %q", metadata.DoltDatabase)
	}

	if metadata.ConnectionMode() != EmbeddedMode {
		t.Errorf("Expected EmbeddedMode, got %v", metadata.ConnectionMode())
	}
}

func TestLoadMetadata_ServerMode(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	metadataJSON := `{
		"database": "dolt",
		"backend": "dolt",
		"dolt_mode": "server",
		"dolt_database": "beads_fo",
		"dolt_server_host": "10.11.0.1",
		"dolt_server_port": 13307,
		"dolt_server_user": "mysql-root"
	}`
	metadataPath := filepath.Join(beadsDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(metadataJSON), 0644); err != nil {
		t.Fatalf("Failed to write metadata.json: %v", err)
	}

	metadata, err := LoadMetadata(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.DoltDatabase != "beads_fo" {
		t.Errorf("Expected DoltDatabase='beads_fo', got %q", metadata.DoltDatabase)
	}

	if metadata.ConnectionMode() != ServerMode {
		t.Errorf("Expected ServerMode, got %v", metadata.ConnectionMode())
	}

	if metadata.ServerHost != "10.11.0.1" {
		t.Errorf("Expected ServerHost='10.11.0.1', got %q", metadata.ServerHost)
	}

	if metadata.ServerPort != 13307 {
		t.Errorf("Expected ServerPort=13307, got %d", metadata.ServerPort)
	}

	if metadata.ServerUser != "mysql-root" {
		t.Errorf("Expected ServerUser='mysql-root', got %q", metadata.ServerUser)
	}
}

func TestLoadMetadata_ServerModeDetectedByFields(t *testing.T) {
	// Server mode should be detected even without explicit dolt_mode field
	// if server connection fields are present
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	metadataJSON := `{
		"database": "dolt",
		"backend": "dolt",
		"dolt_database": "beads_remote",
		"dolt_server_host": "localhost",
		"dolt_server_port": 3307,
		"dolt_server_user": "root"
	}`
	metadataPath := filepath.Join(beadsDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(metadataJSON), 0644); err != nil {
		t.Fatalf("Failed to write metadata.json: %v", err)
	}

	metadata, err := LoadMetadata(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if metadata.ConnectionMode() != ServerMode {
		t.Errorf("Expected ServerMode (detected by fields), got %v", metadata.ConnectionMode())
	}
}

func TestLoadMetadata_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, "nonexistent", ".beads")

	_, err := LoadMetadata(beadsDir)
	if err == nil {
		t.Fatal("Expected error for missing metadata.json")
	}

	if !strings.Contains(err.Error(), "no beads database found") {
		t.Errorf("Error should mention beads database not found, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Is this a beads project?") {
		t.Errorf("Error should suggest running 'bd init', got: %v", err)
	}
}

func TestLoadMetadata_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	metadataPath := filepath.Join(beadsDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write metadata.json: %v", err)
	}

	_, err := LoadMetadata(beadsDir)
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "corrupted or has invalid JSON") {
		t.Errorf("Error should mention invalid JSON, got: %v", err)
	}
}

func TestLoadMetadata_MissingDoltDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	metadataJSON := `{
		"database": "dolt",
		"backend": "dolt"
	}`
	metadataPath := filepath.Join(beadsDir, "metadata.json")
	if err := os.WriteFile(metadataPath, []byte(metadataJSON), 0644); err != nil {
		t.Fatalf("Failed to write metadata.json: %v", err)
	}

	_, err := LoadMetadata(beadsDir)
	if err == nil {
		t.Fatal("Expected error for missing dolt_database")
	}

	if !strings.Contains(err.Error(), "missing required field 'dolt_database'") {
		t.Errorf("Error should mention missing dolt_database, got: %v", err)
	}
}

func TestMetadata_ConnectionMode(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		expected Mode
	}{
		{
			name: "explicit server mode",
			metadata: Metadata{
				DoltMode:     "server",
				DoltDatabase: "test",
			},
			expected: ServerMode,
		},
		{
			name: "server mode by host and port",
			metadata: Metadata{
				DoltDatabase: "test",
				ServerHost:   "localhost",
				ServerPort:   3306,
			},
			expected: ServerMode,
		},
		{
			name: "embedded mode by default",
			metadata: Metadata{
				DoltDatabase: "test",
			},
			expected: EmbeddedMode,
		},
		{
			name: "embedded mode with empty dolt_mode",
			metadata: Metadata{
				DoltMode:     "",
				DoltDatabase: "test",
			},
			expected: EmbeddedMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metadata.ConnectionMode()
			if got != tt.expected {
				t.Errorf("ConnectionMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDoltDir(t *testing.T) {
	beadsDir := "/home/user/project/.beads"
	expected := "/home/user/project/.beads/dolt"
	got := DoltDir(beadsDir)
	if got != expected {
		t.Errorf("DoltDir(%q) = %q, want %q", beadsDir, got, expected)
	}
}

func TestMetadata_ResolveServerPort_AlreadySet(t *testing.T) {
	m := &Metadata{
		DoltDatabase: "test_db",
		ServerPort:   12345,
	}

	port, err := m.ResolveServerPort("/tmp/test")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 12345 {
		t.Errorf("Expected port 12345, got %d", port)
	}

	if m.ServerPort != 12345 {
		t.Errorf("Expected metadata.ServerPort to remain 12345, got %d", m.ServerPort)
	}
}

func TestMetadata_detectPortFromDoltStatus_RunningServer(t *testing.T) {
	// Create a mock bd script that returns "Port: 56789" (running server format)
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	// Create mock bd script
	mockBd := filepath.Join(tmpDir, "bd")
	mockScript := `#!/bin/sh
if [ "$1" = "dolt" ] && [ "$2" = "status" ]; then
    echo "Dolt server: running"
    echo "  Port: 56789"
else
    echo "Unknown command"
    exit 1
fi
`
	if err := os.WriteFile(mockBd, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock bd script: %v", err)
	}

	// Prepend our mock to PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	m := &Metadata{DoltDatabase: "test_db"}
	port, err := m.detectPortFromDoltStatus(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 56789 {
		t.Errorf("Expected port 56789, got %d", port)
	}
}

func TestMetadata_detectPortFromDoltStatus_ExpectedPort(t *testing.T) {
	// Create a mock bd script that returns "Expected port: 54321" (stopped server format)
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	// Create mock bd script
	mockBd := filepath.Join(tmpDir, "bd")
	mockScript := `#!/bin/sh
if [ "$1" = "dolt" ] && [ "$2" = "status" ]; then
    echo "Dolt server: not running"
    echo "  Expected port: 54321"
else
    echo "Unknown command"
    exit 1
fi
`
	if err := os.WriteFile(mockBd, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock bd script: %v", err)
	}

	// Prepend our mock to PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	m := &Metadata{DoltDatabase: "test_db"}
	port, err := m.detectPortFromDoltStatus(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 54321 {
		t.Errorf("Expected port 54321, got %d", port)
	}
}

func TestMetadata_detectPortFromDoltStatus_NoPortFound(t *testing.T) {
	// Create a mock bd script that returns output without port info
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	// Create mock bd script
	mockBd := filepath.Join(tmpDir, "bd")
	mockScript := `#!/bin/sh
if [ "$1" = "dolt" ] && [ "$2" = "status" ]; then
    echo "Dolt server: not running"
    echo "  No database initialized"
else
    echo "Unknown command"
    exit 1
fi
`
	if err := os.WriteFile(mockBd, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock bd script: %v", err)
	}

	// Prepend our mock to PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	m := &Metadata{DoltDatabase: "test_db"}
	port, err := m.detectPortFromDoltStatus(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if port != 0 {
		t.Errorf("Expected port 0 (not found), got %d", port)
	}
}

func TestMetadata_detectPortFromDoltStatus_CommandFails(t *testing.T) {
	// Create a mock bd script that fails
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0750); err != nil {
		t.Fatalf("Failed to create beads dir: %v", err)
	}

	// Create mock bd script that exits with error
	mockBd := filepath.Join(tmpDir, "bd")
	mockScript := `#!/bin/sh
echo "Error: something went wrong" >&2
exit 1
`
	if err := os.WriteFile(mockBd, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create mock bd script: %v", err)
	}

	// Prepend our mock to PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	m := &Metadata{DoltDatabase: "test_db"}
	port, err := m.detectPortFromDoltStatus(beadsDir)
	if err != nil {
		t.Fatalf("Expected no error when command fails (graceful fallback), got: %v", err)
	}

	if port != 0 {
		t.Errorf("Expected port 0 when command fails, got %d", port)
	}
}

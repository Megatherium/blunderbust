// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTUIConfig_WithValues(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	yamlContent := `
filepicker_recents:
  - /path/to/one
  - /path/to/two
filepicker_max_recents: 10
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if len(cfg.FilePickerRecents) != 2 {
		t.Errorf("Expected 2 recents, got %d", len(cfg.FilePickerRecents))
	}

	if cfg.FilePickerRecents[0] != "/path/to/one" {
		t.Errorf("Expected first recent to be '/path/to/one', got '%s'", cfg.FilePickerRecents[0])
	}

	if cfg.FilePickerMaxRecents != 10 {
		t.Errorf("Expected FilePickerMaxRecents to be 10, got %d", cfg.FilePickerMaxRecents)
	}
}

func TestLoadTUIConfig_DefaultMaxRecents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	yamlContent := `
filepicker_recents:
  - /path/to/one
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if cfg.FilePickerMaxRecents != DefaultMaxRecents {
		t.Errorf("Expected FilePickerMaxRecents default to be %d, got %d", DefaultMaxRecents, cfg.FilePickerMaxRecents)
	}
}

func TestLoadTUIConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	yamlContent := ``

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if cfg.FilePickerRecents != nil {
		t.Error("Expected FilePickerRecents to be nil")
	}

	if cfg.FilePickerMaxRecents != DefaultMaxRecents {
		t.Errorf("Expected FilePickerMaxRecents default to be %d, got %d", DefaultMaxRecents, cfg.FilePickerMaxRecents)
	}
}

func TestLoadTUIConfig_ZeroMaxRecents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	yamlContent := `
filepicker_max_recents: 0
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if cfg.FilePickerMaxRecents != DefaultMaxRecents {
		t.Errorf("Expected FilePickerMaxRecents default to be %d when 0, got %d", DefaultMaxRecents, cfg.FilePickerMaxRecents)
	}
}

func TestLoadTUIConfig_NegativeMaxRecents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	yamlContent := `
filepicker_max_recents: -5
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if cfg.FilePickerMaxRecents != DefaultMaxRecents {
		t.Errorf("Expected FilePickerMaxRecents default to be %d when negative, got %d", DefaultMaxRecents, cfg.FilePickerMaxRecents)
	}
}

func TestSaveTUIConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	cfg := &TUIConfig{
		FilePickerRecents:    []string{"/path/one", "/path/two"},
		FilePickerMaxRecents: 15,
	}

	if err := SaveTUIConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveTUIConfig failed: %v", err)
	}

	loadedConfig, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if len(loadedConfig.FilePickerRecents) != 2 {
		t.Errorf("Expected 2 recents after save/load, got %d", len(loadedConfig.FilePickerRecents))
	}

	if loadedConfig.FilePickerRecents[0] != "/path/one" {
		t.Errorf("Expected first recent to be '/path/one', got '%s'", loadedConfig.FilePickerRecents[0])
	}

	if loadedConfig.FilePickerMaxRecents != 15 {
		t.Errorf("Expected FilePickerMaxRecents to be 15, got %d", loadedConfig.FilePickerMaxRecents)
	}
}

func TestSaveTUIConfig_ZeroMaxRecents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	cfg := &TUIConfig{
		FilePickerRecents:    []string{"/path/one"},
		FilePickerMaxRecents: 0,
	}

	if err := SaveTUIConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveTUIConfig failed: %v", err)
	}

	loadedConfig, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if loadedConfig.FilePickerMaxRecents != DefaultMaxRecents {
		t.Errorf("Expected FilePickerMaxRecents to be %d after saving 0, got %d", DefaultMaxRecents, loadedConfig.FilePickerMaxRecents)
	}
}

func TestLoadTUIConfig_FileNotFound(t *testing.T) {
	configPath := "/nonexistent/path/tui_config.yaml"

	_, err := LoadTUIConfig(configPath)
	if err == nil {
		t.Fatal("Expected error when loading nonexistent file")
	}

	if !os.IsNotExist(err) && !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected file not found error, got: %v", err)
	}
}

func TestLoadTUIConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	invalidYAML := `
filepicker_recents:
  - /path/one
  unclosed bracket: [
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadTUIConfig(configPath)
	if err == nil {
		t.Fatal("Expected error when loading invalid YAML")
	}
}

func TestTUIConfig_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	original := &TUIConfig{
		FilePickerRecents:    []string{"/home/user/project1", "/home/user/project2"},
		FilePickerMaxRecents: 8,
	}

	if err := SaveTUIConfig(configPath, original); err != nil {
		t.Fatalf("Failed to save TUI config: %v", err)
	}

	loaded, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load TUI config: %v", err)
	}

	if len(loaded.FilePickerRecents) != len(original.FilePickerRecents) {
		t.Errorf("Recents count mismatch: expected %d, got %d", len(original.FilePickerRecents), len(loaded.FilePickerRecents))
	}

	for i, recent := range original.FilePickerRecents {
		if loaded.FilePickerRecents[i] != recent {
			t.Errorf("Recent at index %d mismatch: expected '%s', got '%s'", i, recent, loaded.FilePickerRecents[i])
		}
	}

	if loaded.FilePickerMaxRecents != original.FilePickerMaxRecents {
		t.Errorf("FilePickerMaxRecents mismatch: expected %d, got %d", original.FilePickerMaxRecents, loaded.FilePickerMaxRecents)
	}
}

func TestSaveTUIConfig_EmptyRecents(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "tui_config.yaml")

	cfg := &TUIConfig{
		FilePickerRecents:    []string{},
		FilePickerMaxRecents: 5,
	}

	if err := SaveTUIConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveTUIConfig failed: %v", err)
	}

	loadedConfig, err := LoadTUIConfig(configPath)
	if err != nil {
		t.Fatalf("LoadTUIConfig failed: %v", err)
	}

	if loadedConfig.FilePickerRecents != nil && len(loadedConfig.FilePickerRecents) != 0 {
		t.Errorf("Expected 0 recents, got %d", len(loadedConfig.FilePickerRecents))
	}
}

func TestDefaultMaxRecents(t *testing.T) {
	if DefaultMaxRecents != 5 {
		t.Errorf("Expected DefaultMaxRecents to be 5, got %d", DefaultMaxRecents)
	}
}

// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// yamlTUIConfig is the raw YAML structure for TUI-specific configuration.
type yamlTUIConfig struct {
	FilePickerRecents    []string `yaml:"filepicker_recents,omitempty"`
	FilePickerMaxRecents int      `yaml:"filepicker_max_recents,omitempty"`
}

// LoadTUIConfig reads and parses a TUI YAML configuration file.
// Returns actionable errors for parse errors or file not found.
// If FilePickerMaxRecents is not specified or invalid, defaults to DefaultMaxRecents.
func LoadTUIConfig(path string) (*TUIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("TUI config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read TUI config file %s: %w", path, err)
	}

	var raw yamlTUIConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse TUI YAML in %s: %w", path, err)
	}

	cfg := &TUIConfig{
		FilePickerRecents: raw.FilePickerRecents,
	}

	// Default FilePickerMaxRecents to DefaultMaxRecents if not specified or invalid
	if raw.FilePickerMaxRecents > 0 {
		cfg.FilePickerMaxRecents = raw.FilePickerMaxRecents
	} else {
		cfg.FilePickerMaxRecents = DefaultMaxRecents
	}

	return cfg, nil
}

// SaveTUIConfig writes the TUI configuration to a YAML file.
func SaveTUIConfig(path string, cfg *TUIConfig) error {
	yamlCfg := yamlTUIConfig{
		FilePickerRecents: cfg.FilePickerRecents,
	}

	if cfg.FilePickerMaxRecents > 0 {
		yamlCfg.FilePickerMaxRecents = cfg.FilePickerMaxRecents
	}

	data, err := yaml.Marshal(yamlCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal TUI config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write TUI config file: %w", err)
	}

	return nil
}

// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/megatherium/blunderbust/internal/domain"
	"gopkg.in/yaml.v3"
)

// loadTemplateValue loads a template value from a file if it starts with '@'.
// If the value doesn't start with '@', returns it as-is.
// Returns an actionable error if the file cannot be read.
func loadTemplateValue(value, configDir string) (string, error) {
	if !strings.HasPrefix(value, "@") {
		return value, nil
	}

	filePath := strings.TrimPrefix(value, "@")
	resolvedPath := filePath
	if !filepath.IsAbs(filePath) {
		resolvedPath = filepath.Join(configDir, filePath)
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("failed to load template file: %s (file not found)", value)
		}
		return "", fmt.Errorf("failed to load template file: %s: %w", value, err)
	}

	return string(content), nil
}

// Load reads and parses a YAML configuration file.
// Returns actionable errors for missing fields, parse errors, or file not found.
func (l *YAMLLoader) Load(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var raw yamlConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	configDir := filepath.Dir(path)
	return l.convertAndValidate(&raw, configDir)
}

// validateHarnessNames checks for duplicate harness names.
func (l *YAMLLoader) validateHarnessNames(harnesses []yamlHarness) error {
	seenNames := make(map[string]int)
	for i, rawHarness := range harnesses {
		name := rawHarness.Name
		if name == "" {
			continue
		}
		if firstIdx, exists := seenNames[name]; exists {
			return fmt.Errorf("duplicate harness name %q at index %d (first defined at index %d)", name, i, firstIdx)
		}
		seenNames[name] = i
	}
	return nil
}

// parseWorkspace parses and validates workspace projects.
func (l *YAMLLoader) parseWorkspace(defaultWorkspace yamlWorkspace, configDir string) ([]domain.Project, error) {
	var projects []domain.Project
	seenDirs := make(map[string]bool)

	for _, p := range defaultWorkspace.Projects {
		if p.Dir == "" {
			return nil, fmt.Errorf("project must specify a directory")
		}

		projectDir := p.Dir
		if !filepath.IsAbs(projectDir) {
			projectDir = filepath.Join(configDir, projectDir)
		}

		info, err := os.Stat(projectDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
			}
			return nil, fmt.Errorf("error checking project directory %s: %w", projectDir, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("project path is not a directory: %s", projectDir)
		}

		cleanDir := filepath.Clean(projectDir)
		if seenDirs[cleanDir] {
			return nil, fmt.Errorf("duplicate project directory: %s", cleanDir)
		}
		seenDirs[cleanDir] = true

		name := p.Name
		if name == "" {
			name = filepath.Base(p.Dir)
		}
		projects = append(projects, domain.Project{
			Dir:  cleanDir,
			Name: name,
		})
	}
	return projects, nil
}

// convertAndValidate converts the raw YAML to domain types and validates.
func (l *YAMLLoader) convertAndValidate(raw *yamlConfig, configDir string) (*domain.Config, error) {
	if len(raw.Harnesses) == 0 {
		return nil, fmt.Errorf("config must define at least one harness")
	}
	if err := l.validateHarnessNames(raw.Harnesses); err != nil {
		return nil, err
	}

	config := &domain.Config{
		Harnesses: make([]domain.Harness, 0, len(raw.Harnesses)),
	}

	for i, rawHarness := range raw.Harnesses {
		harness, err := l.convertHarness(rawHarness, i, configDir)
		if err != nil {
			return nil, err
		}
		config.Harnesses = append(config.Harnesses, *harness)
	}

	if defaultWorkspace, ok := raw.Workspaces["default"]; ok {
		projects, err := l.parseWorkspace(defaultWorkspace, configDir)
		if err != nil {
			return nil, err
		}
		config.Workspace = domain.Workspace{
			Name:     "default",
			Projects: projects,
		}
	}

	if raw.Launcher != nil {
		launcherConfig, err := l.convertLauncherConfig(raw.Launcher)
		if err != nil {
			return nil, err
		}
		config.Launcher = launcherConfig
	} else {
		config.Launcher = &domain.LauncherConfig{Target: "foreground"}
	}

	if raw.Defaults != nil {
		config.Defaults = &domain.Defaults{
			Harness: raw.Defaults.Harness,
			Model:   raw.Defaults.Model,
			Agent:   raw.Defaults.Agent,
		}
	}

	autostart := true
	if raw.General != nil && raw.General.AutostartDolt != nil {
		autostart = *raw.General.AutostartDolt
	}
	config.General = &domain.GeneralConfig{AutostartDolt: autostart}

	return config, nil
}

// convertLauncherConfig validates and converts launcher configuration.
func (l *YAMLLoader) convertLauncherConfig(raw *yamlLauncherConfig) (*domain.LauncherConfig, error) {
	target := strings.ToLower(raw.Target)
	if target == "" {
		target = "foreground"
	}
	if target != "foreground" && target != "background" {
		return nil, fmt.Errorf("invalid launcher.target value: %q (must be 'foreground' or 'background')", raw.Target)
	}
	return &domain.LauncherConfig{Target: target}, nil
}

// convertHarness validates and converts a single YAML harness to domain type.
func (l *YAMLLoader) convertHarness(raw yamlHarness, index int, configDir string) (*domain.Harness, error) {
	harnessName := raw.Name
	if harnessName == "" {
		return nil, fmt.Errorf("harness at index %d is missing required field: name", index)
	}

	commandTemplate, err := loadTemplateValue(raw.CommandTemplate, configDir)
	if err != nil {
		return nil, fmt.Errorf("harness %q: %w", harnessName, err)
	}
	if commandTemplate == "" {
		return nil, fmt.Errorf("harness %q is missing required field: command_template", harnessName)
	}

	promptTemplate, err := loadTemplateValue(raw.PromptTemplate, configDir)
	if err != nil {
		return nil, fmt.Errorf("harness %q: %w", harnessName, err)
	}

	models := raw.Models
	if models == nil {
		models = []string{}
	}

	agents := raw.Agents
	if agents == nil {
		agents = []string{}
	}

	env := raw.Env
	if env == nil {
		env = map[string]string{}
	}

	return &domain.Harness{
		Name:            harnessName,
		CommandTemplate: commandTemplate,
		PromptTemplate:  promptTemplate,
		SupportedModels: models,
		SupportedAgents: agents,
		Env:             env,
	}, nil
}

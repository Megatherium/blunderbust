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

// yamlConfig is the raw YAML structure for unmarshaling.
type yamlConfig struct {
	Harnesses  []yamlHarness            `yaml:"harnesses"`
	Launcher   *yamlLauncherConfig      `yaml:"launcher,omitempty"`
	Defaults   *yamlDefaults            `yaml:"defaults,omitempty"`
	General    *yamlGeneralConfig       `yaml:"general,omitempty"`
	Workspaces map[string]yamlWorkspace `yaml:"workspaces,omitempty"`
}

type yamlWorkspace struct {
	Projects []yamlProject `yaml:"projects"`
}

type yamlProject struct {
	Dir  string `yaml:"dir"`
	Name string `yaml:"name,omitempty"`
}

// yamlLauncherConfig is the raw YAML structure for launcher configuration.
type yamlLauncherConfig struct {
	Target string `yaml:"target,omitempty"`
}

// yamlHarness is the raw YAML structure for a harness definition.
type yamlHarness struct {
	Name            string            `yaml:"name"`
	CommandTemplate string            `yaml:"command_template"`
	PromptTemplate  string            `yaml:"prompt_template,omitempty"`
	Models          []string          `yaml:"models,omitempty"`
	Agents          []string          `yaml:"agents,omitempty"`
	Env             map[string]string `yaml:"env,omitempty"`
}

// yamlDefaults is the raw YAML structure for default settings.
type yamlDefaults struct {
	Harness string `yaml:"harness,omitempty"`
	Model   string `yaml:"model,omitempty"`
	Agent   string `yaml:"agent,omitempty"`
}

// yamlGeneralConfig is the raw YAML structure for general settings.
type yamlGeneralConfig struct {
	AutostartDolt *bool `yaml:"autostart_dolt,omitempty"`
}

// YAMLLoader implements the Loader interface for YAML configuration files.
type YAMLLoader struct{}

// NewYAMLLoader creates a new YAML configuration loader.
func NewYAMLLoader() *YAMLLoader {
	return &YAMLLoader{}
}

// loadTemplateValue loads a template value from a file if it starts with '@'.
// If the value doesn't start with '@', returns it as-is.
// Returns an actionable error if the file cannot be read.
func loadTemplateValue(value, configDir string) (string, error) {
	if !strings.HasPrefix(value, "@") {
		return value, nil
	}

	filePath := strings.TrimPrefix(value, "@")
	var resolvedPath string

	// If the path starts with "/", it's an absolute path - use it as-is
	// Otherwise, treat it as relative to the config directory
	if filepath.IsAbs(filePath) {
		resolvedPath = filePath
	} else {
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

// convertAndValidate converts the raw YAML to domain types and validates.
func (l *YAMLLoader) convertAndValidate(raw *yamlConfig, configDir string) (*domain.Config, error) {
	if len(raw.Harnesses) == 0 {
		return nil, fmt.Errorf("config must define at least one harness")
	}

	seenNames := make(map[string]int)
	for i, rawHarness := range raw.Harnesses {
		name := rawHarness.Name
		if name == "" {
			continue
		}
		if firstIdx, exists := seenNames[name]; exists {
			return nil, fmt.Errorf("duplicate harness name %q at index %d (first defined at index %d)", name, i, firstIdx)
		}
		seenNames[name] = i
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
		var projects []domain.Project
		seenDirs := make(map[string]bool)

		for _, p := range defaultWorkspace.Projects {
			if p.Dir == "" {
				return nil, fmt.Errorf("project must specify a directory")
			}

			// Validate directory exists
			info, err := os.Stat(p.Dir)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, fmt.Errorf("project directory does not exist: %s", p.Dir)
				}
				return nil, fmt.Errorf("error checking project directory %s: %w", p.Dir, err)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("project path is not a directory: %s", p.Dir)
			}

			// Validate directory uniqueness
			cleanDir := filepath.Clean(p.Dir)
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
		config.Workspace = domain.Workspace{
			Name:     "default",
			Projects: projects,
		}
	} else {
		// Backward compatibility: handled by the caller/App using beadsDir if Workspace.Projects is empty
	}

	if raw.Launcher != nil {
		launcherConfig, err := l.convertLauncherConfig(raw.Launcher)
		if err != nil {
			return nil, err
		}
		config.Launcher = launcherConfig
	} else {
		// Default to foreground mode if not specified
		config.Launcher = &domain.LauncherConfig{
			Target: "foreground",
		}
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
	config.General = &domain.GeneralConfig{
		AutostartDolt: autostart,
	}

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

	return &domain.LauncherConfig{
		Target: target,
	}, nil
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

	// Initialize empty slices to avoid nil checks downstream
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

// Compile-time check that YAMLLoader implements Loader interface.
var _ Loader = (*YAMLLoader)(nil)

// Save writes the configuration to a YAML file.
func (l *YAMLLoader) Save(path string, cfg *domain.Config) error {
	yamlCfg := l.domainToYAML(cfg)

	data, err := yaml.Marshal(yamlCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// domainToYAML converts domain.Config to yamlConfig for YAML marshaling.
func (l *YAMLLoader) domainToYAML(cfg *domain.Config) yamlConfig {
	var yamlCfg yamlConfig

	// Convert harnesses
	if len(cfg.Harnesses) > 0 {
		yamlCfg.Harnesses = make([]yamlHarness, len(cfg.Harnesses))
		for i, h := range cfg.Harnesses {
			yamlCfg.Harnesses[i] = yamlHarness{
				Name:            h.Name,
				CommandTemplate: h.CommandTemplate,
				PromptTemplate:  h.PromptTemplate,
				Models:          h.SupportedModels,
				Agents:          h.SupportedAgents,
				Env:             h.Env,
			}
		}
	}

	// Convert launcher
	if cfg.Launcher != nil {
		yamlCfg.Launcher = &yamlLauncherConfig{
			Target: cfg.Launcher.Target,
		}
	}

	// Convert defaults
	if cfg.Defaults != nil {
		yamlCfg.Defaults = &yamlDefaults{
			Harness: cfg.Defaults.Harness,
			Model:   cfg.Defaults.Model,
			Agent:   cfg.Defaults.Agent,
		}
	}

	// Convert general config
	if cfg.General != nil {
		autostart := cfg.General.AutostartDolt
		yamlCfg.General = &yamlGeneralConfig{
			AutostartDolt: &autostart,
		}
	}

	// Convert workspace
	if len(cfg.Workspace.Projects) > 0 {
		projects := make([]yamlProject, len(cfg.Workspace.Projects))
		for i, p := range cfg.Workspace.Projects {
			projects[i] = yamlProject{
				Dir:  p.Dir,
				Name: p.Name,
			}
		}
		yamlCfg.Workspaces = map[string]yamlWorkspace{
			"default": {Projects: projects},
		}
	}

	return yamlCfg
}

// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

// yamlConfig is the raw YAML structure for unmarshaling.
type yamlConfig struct {
	Harnesses            []yamlHarness            `yaml:"harnesses"`
	Launcher             *yamlLauncherConfig      `yaml:"launcher,omitempty"`
	Defaults             *yamlDefaults            `yaml:"defaults,omitempty"`
	General              *yamlGeneralConfig       `yaml:"general,omitempty"`
	Workspaces           map[string]yamlWorkspace `yaml:"workspaces,omitempty"`
	FilePickerRecents    []string                 `yaml:"filepicker_recents,omitempty"`
	FilePickerMaxRecents int                      `yaml:"filepicker_max_recents,omitempty"`
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

// Compile-time check that YAMLLoader implements Loader interface.
var _ Loader = (*YAMLLoader)(nil)

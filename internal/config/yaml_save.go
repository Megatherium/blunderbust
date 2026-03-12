// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package config

import (
	"fmt"
	"os"

	"github.com/megatherium/blunderbust/internal/domain"
	"gopkg.in/yaml.v3"
)

// Save writes the configuration to a YAML file.
func (l *YAMLLoader) Save(path string, cfg *domain.Config) error {
	yamlCfg := l.domainToYAML(cfg)

	data, err := yaml.Marshal(yamlCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// domainToYAML converts domain.Config to yamlConfig for YAML marshaling.
func (l *YAMLLoader) domainToYAML(cfg *domain.Config) yamlConfig {
	var yamlCfg yamlConfig

	if len(cfg.Harnesses) > 0 {
		yamlCfg.Harnesses = make([]yamlHarness, len(cfg.Harnesses))
		for i, harness := range cfg.Harnesses {
			yamlCfg.Harnesses[i] = yamlHarness{
				Name:            harness.Name,
				CommandTemplate: harness.CommandTemplate,
				PromptTemplate:  harness.PromptTemplate,
				Models:          harness.SupportedModels,
				Agents:          harness.SupportedAgents,
				Env:             harness.Env,
			}
		}
	}

	if cfg.Launcher != nil {
		yamlCfg.Launcher = &yamlLauncherConfig{Target: cfg.Launcher.Target}
	}

	if cfg.Defaults != nil {
		yamlCfg.Defaults = &yamlDefaults{
			Harness: cfg.Defaults.Harness,
			Model:   cfg.Defaults.Model,
			Agent:   cfg.Defaults.Agent,
		}
	}

	if cfg.General != nil {
		autostart := cfg.General.AutostartDolt
		yamlCfg.General = &yamlGeneralConfig{
			AutostartDolt: &autostart,
		}
	}

	if len(cfg.Workspace.Projects) > 0 {
		projects := make([]yamlProject, len(cfg.Workspace.Projects))
		for i, project := range cfg.Workspace.Projects {
			projects[i] = yamlProject{
				Dir:  project.Dir,
				Name: project.Name,
			}
		}
		yamlCfg.Workspaces = map[string]yamlWorkspace{
			"default": {Projects: projects},
		}
	}

	return yamlCfg
}

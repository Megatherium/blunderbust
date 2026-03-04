package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/stretchr/testify/assert"
)

func TestRenderFilePicker_ContainsTitle(t *testing.T) {
	fp := filepicker.New()
	cfg := FilePickerConfig{
		Filepicker: fp,
		Theme:      MatrixTheme,
	}

	s := RenderFilePicker(cfg)
	assert.Contains(t, s, "Add Project - Select Directory")
}

func TestRenderFilePicker_ContainsHelpText(t *testing.T) {
	fp := filepicker.New()
	cfg := FilePickerConfig{
		Filepicker: fp,
		Theme:      MatrixTheme,
	}

	s := RenderFilePicker(cfg)
	assert.Contains(t, s, "Press 'a' to select highlighted directory")
	assert.Contains(t, s, "esc' to cancel")
}

func TestRenderFilePicker_RendersFilepicker(t *testing.T) {
	fp := filepicker.New()
	cfg := FilePickerConfig{
		Filepicker: fp,
		Theme:      MatrixTheme,
	}

	// Just verify it renders without error
	s := RenderFilePicker(cfg)
	assert.NotEmpty(t, s)
}

func TestRenderAddProjectModal_ContainsTitle(t *testing.T) {
	cfg := AddProjectConfig{
		PendingProjectPath: "/path/to/project",
		Theme:              MatrixTheme,
	}

	s := RenderAddProjectModal(cfg)
	assert.Contains(t, s, "Add Project?")
}

func TestRenderAddProjectModal_ContainsProjectPath(t *testing.T) {
	cfg := AddProjectConfig{
		PendingProjectPath: "/path/to/project",
		Theme:              MatrixTheme,
	}

	s := RenderAddProjectModal(cfg)
	assert.Contains(t, s, "/path/to/project")
}

func TestRenderAddProjectModal_ContainsConfirmationOptions(t *testing.T) {
	cfg := AddProjectConfig{
		PendingProjectPath: "/path/to/project",
		Theme:              MatrixTheme,
	}

	s := RenderAddProjectModal(cfg)
	assert.Contains(t, s, "Press 'y' or Enter to confirm")
	assert.Contains(t, s, "'n' or Esc to cancel")
}

func TestRenderAddProjectModal_UsesThemeColors(t *testing.T) {
	customTheme := ThemePalette{
		Name:       "Custom",
		TitleColor: "#123456",
		ReadyColor: "#654321",
	}

	cfg := AddProjectConfig{
		PendingProjectPath: "/path/to/project",
		Theme:              customTheme,
	}

	// Just verify it renders without error with custom theme
	s := RenderAddProjectModal(cfg)
	assert.NotEmpty(t, s)
	assert.Contains(t, s, "Add Project?")
	assert.Contains(t, s, "/path/to/project")
}

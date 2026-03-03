package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/lipgloss"
)

// FilePickerConfig holds configuration for rendering the file picker view
type FilePickerConfig struct {
	Filepicker filepicker.Model
	Theme      *ThemePalette
}

// RenderFilePicker renders the file picker for adding projects
func RenderFilePicker(cfg FilePickerConfig) string {
	var s strings.Builder

	theme := cfg.Theme
	if theme == nil {
		theme = &MatrixTheme
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.TitleColor).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	s.WriteString(titleStyle.Render("Add Project - Select Directory"))
	s.WriteString("\n\n")
	s.WriteString(cfg.Filepicker.View())
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Press 'a' to select highlighted directory, 'esc' to cancel"))

	return s.String()
}

// AddProjectConfig holds configuration for rendering the add project modal
type AddProjectConfig struct {
	PendingProjectPath string
	Theme              *ThemePalette
}

// RenderAddProjectModal renders the confirmation modal for adding a project
func RenderAddProjectModal(cfg AddProjectConfig) string {
	var s strings.Builder

	theme := cfg.Theme
	if theme == nil {
		theme = &MatrixTheme
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.TitleColor).
		MarginBottom(1)

	pathStyle := lipgloss.NewStyle().
		Foreground(theme.ReadyColor).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Faint(true).
		MarginTop(1)

	s.WriteString(titleStyle.Render("Add Project?"))
	s.WriteString("\n\n")
	fmt.Fprintf(&s, "Add project at:\n%s", pathStyle.Render(cfg.PendingProjectPath))
	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("Press 'y' or Enter to confirm, 'n' or Esc to cancel"))

	return s.String()
}

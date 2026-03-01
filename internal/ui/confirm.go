package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/domain"
)

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(GradientColors[8])).MarginBottom(1)
	itemStyle        = lipgloss.NewStyle().MarginLeft(2)
	dryRunBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFF")).
				Background(lipgloss.Color("#FF6B6B")).
				Padding(0, 1).
				MarginBottom(1)
)

func confirmView(selection domain.Selection, renderer *config.Renderer, dryRun bool, workDir string, theme *ThemePalette) string {
	// Use Matrix theme as default if nil
	if theme == nil {
		theme = &MatrixTheme
	}

	// Arcade-style styles using theme colors
	readyTextStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ReadyColor).
		MarginTop(1).
		MarginBottom(1)

	launchButtonStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.LaunchFg).
		Background(theme.LaunchBg).
		Padding(0, 4).
		Width(20).
		Align(lipgloss.Center)

	// Update title style with theme color
	themeTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.TitleColor).
		MarginBottom(1)

	s := ""

	if dryRun {
		s += dryRunBadgeStyle.Render("[DRY RUN]") + "\n"
	}

	s += themeTitleStyle.Render("Confirm Launch Spec") + "\n"
	s += fmt.Sprintf("Ticket:  %s (%s)\n", itemStyle.Render(selection.Ticket.ID), selection.Ticket.Title)
	s += fmt.Sprintf("Harness: %s\n", itemStyle.Render(selection.Harness.Name))

	modelName := selection.Model
	if modelName == "" {
		modelName = "None"
	}
	s += fmt.Sprintf("Model:   %s\n", itemStyle.Render(modelName))

	agentName := selection.Agent
	if agentName == "" {
		agentName = "None"
	}
	s += fmt.Sprintf("Agent:   %s\n\n", itemStyle.Render(agentName))

	if workDir != "" {
		s += fmt.Sprintf("WorkDir: %s\n\n", itemStyle.Render(workDir))
	}

	if renderer != nil {
		spec, err := renderer.RenderSelection(selection, workDir)
		if err == nil && spec != nil {
			s += themeTitleStyle.Render("Rendered Command:") + "\n"
			s += itemStyle.Render(fmt.Sprintf("```bash\n%s\n```", spec.RenderedCommand)) + "\n\n"
			if spec.RenderedPrompt != "" {
				s += themeTitleStyle.Render("Rendered Prompt:") + "\n"
				promptLines := strings.Split(spec.RenderedPrompt, "\n")
				for _, line := range promptLines {
					s += itemStyle.Render(line) + "\n"
				}
				s += "\n"
			}
		} else {
			s += itemStyle.Render(fmt.Sprintf("Error rendering: %v", err)) + "\n\n"
		}
	}

	// Arcade-style ready indicator
	s += readyTextStyle.Render("READY?") + "\n"

	// Big launch button
	s += launchButtonStyle.Render("LAUNCH") + "\n\n"

	s += "[Press Enter to launch, esc to go back]"
	return s
}

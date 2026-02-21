package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/megatherium/blunderbuss/internal/config"
	"github.com/megatherium/blunderbuss/internal/domain"
)

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	itemStyle        = lipgloss.NewStyle().MarginLeft(2)
	dryRunBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFF")).
				Background(lipgloss.Color("#FF6B6B")).
				Padding(0, 1).
				MarginBottom(1)
)

func confirmView(selection domain.Selection, renderer *config.Renderer, dryRun bool) string {
	s := ""

	if dryRun {
		s += dryRunBadgeStyle.Render("[DRY RUN]") + "\n"
	}

	s += titleStyle.Render("Confirm Launch Spec") + "\n"
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

	if renderer != nil {
		spec, err := renderer.RenderSelection(selection)
		if err == nil && spec != nil {
			s += titleStyle.Render("Rendered Command:") + "\n"
			s += itemStyle.Render(fmt.Sprintf("```bash\n%s\n```", spec.RenderedCommand)) + "\n\n"
			if spec.RenderedPrompt != "" {
				s += titleStyle.Render("Rendered Prompt:") + "\n"
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

	s += "[Press Enter to launch, esc to go back]"
	return s
}

package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/megatherium/blunderbuss/internal/domain"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	itemStyle  = lipgloss.NewStyle().MarginLeft(2)
)

func confirmView(selection domain.Selection) string {
	s := titleStyle.Render("Confirm Launch Spec") + "\n"
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

	s += "[Press Enter to launch, esc to go back]"
	return s
}

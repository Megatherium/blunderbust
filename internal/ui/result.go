package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/megatherium/blunderbust/internal/domain"
)

func resultView(m UIModel, res *domain.LaunchResult, err error, statusEmoji, status string) string {
	s := titleStyle.Render("Launch Result") + "\n\n"
	switch {
	case err != nil:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", err))
	case res != nil && res.Error != nil:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Launch Error: %v", res.Error))
	case res != nil:
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("Success!") + "\n"
		s += fmt.Sprintf("Window: %s\n", res.WindowName)
		if statusEmoji != "" && status != "" {
			s += fmt.Sprintf("Status: %s %s\n", statusEmoji, status)
		}
		// Add viewport for output streaming
		s += "\n" + lipgloss.NewStyle().Bold(true).Render("Output:") + "\n"
		s += m.viewport.View()
	}
	s += "\n[Press q to quit]"
	return s
}

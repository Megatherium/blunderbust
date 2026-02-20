package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/megatherium/blunderbuss/internal/domain"
)

func resultView(res *domain.LaunchResult, err error) string {
	s := titleStyle.Render("Launch Result") + "\n\n"
	if err != nil {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", err))
	} else if res != nil && res.Error != nil {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Launch Error: %v", res.Error))
	} else if res != nil {
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("Success!") + "\n"
		s += fmt.Sprintf("Window: %s\n", res.WindowName)
	}
	s += "\n[Press q to quit]"
	return s
}

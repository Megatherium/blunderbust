package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/megatherium/blunderbust/internal/data/dolt"
)

var (
	errorTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6666"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(2)
)

// errorView renders an error screen with actionable messaging.
func errorView(err error) string {
	if err == nil {
		return "An unknown error occurred"
	}

	errStr := err.Error()
	var b strings.Builder

	b.WriteString(errorTitleStyle.Render("Error"))
	b.WriteString("\n\n")

	// Provide user-friendly messages based on error type
	switch {
	case strings.Contains(errStr, "metadata.json is missing"):
		b.WriteString(errorStyle.Render("No beads database found."))
		b.WriteString("\n\n")
		b.WriteString("Is this a beads project? Run 'bd init' to initialize beads in this repository.")

	case strings.Contains(errStr, "dolt database directory not found"):
		b.WriteString(errorStyle.Render("The beads database is not initialized."))
		b.WriteString("\n\n")
		b.WriteString("Run 'bd init' to create the beads database.")

	case strings.Contains(errStr, "failed to connect to") && strings.Contains(errStr, "server"):
		b.WriteString(errorStyle.Render("Cannot connect to Dolt server."))
		b.WriteString("\n\n")
		b.WriteString("Please check that the Dolt server is running and the connection details are correct.")

	case dolt.IsErrServerNotRunning(err):
		b.WriteString(errorStyle.Render("Dolt server is not running."))
		b.WriteString("\n\n")
		b.WriteString("Start dolt server? [y/N]")

	case strings.Contains(errStr, "connection refused"):
		b.WriteString(errorStyle.Render("Connection refused."))
		b.WriteString("\n\n")
		b.WriteString("Cannot connect to the database server. Please check that it's running.")

	default:
		// Show the original error for unknown cases
		b.WriteString(errorStyle.Render(errStr))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("\nPress 'q' to quit."))

	return b.String()
}

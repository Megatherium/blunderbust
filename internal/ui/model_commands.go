package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
)

func loadTicketsCmd(store data.TicketStore) tea.Cmd {
	return func() tea.Msg {
		tickets, err := store.ListTickets(context.Background(), data.TicketFilter{})
		if err != nil {
			return errMsg{err}
		}
		return ticketsLoadedMsg(tickets)
	}
}

func loadModalCmd(ticketID string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("bd", "show", ticketID).CombinedOutput()
		if err != nil {
			return modalContentMsg(fmt.Sprintf("Error loading bd show:\n%v\n%s", err, string(out)))
		}
		return modalContentMsg(string(out))
	}
}

// extractRepoRoot extracts the repository root path from a beadsDir path.
// It handles both "/path/to/.beads" and "/path/to/.beads/" patterns.
func extractRepoRoot(beadsDir string) string {
	repoRoot := beadsDir
	if idx := strings.LastIndex(beadsDir, "/.beads"); idx > 0 {
		repoRoot = beadsDir[:idx]
	} else if strings.HasSuffix(beadsDir, ".beads") {
		repoRoot = filepath.Dir(beadsDir)
	}
	return repoRoot
}

func discoverWorktreesCmd(beadsDir string) tea.Cmd {
	return func() tea.Msg {
		repoRoot := extractRepoRoot(beadsDir)

		absRepoRoot, err := filepath.Abs(repoRoot)
		if err != nil {
			return worktreesDiscoveredMsg{err: fmt.Errorf("failed to resolve repo root: %w", err)}
		}

		discoverer := data.NewWorktreeDiscoverer(absRepoRoot)
		worktrees, err := discoverer.Discover(context.Background())
		if err != nil {
			return worktreesDiscoveredMsg{err: err}
		}

		projectName := data.GetProjectName(absRepoRoot)
		nodes := discoverer.BuildSidebarTree(worktrees, projectName)
		return worktreesDiscoveredMsg{nodes: nodes}
	}
}

func (m UIModel) launchCmd() tea.Cmd {
	return func() tea.Msg {
		workDir := m.selectedWorktree
		if workDir == "" {
			workDir = extractRepoRoot(m.app.opts.BeadsDir)
		}

		spec, err := m.app.Renderer.RenderSelection(m.selection, workDir)
		if err != nil {
			return launchResultMsg{
				res: nil,
				err: fmt.Errorf("failed to render launch spec: %w", err),
			}
		}

		spec.WindowName = m.selection.Ticket.ID

		res, err := m.app.launcher.Launch(context.Background(), *spec)
		return launchResultMsg{res: res, err: err}
	}
}

func (m UIModel) pollWindowStatusCmd(windowName string) tea.Cmd {
	return func() tea.Msg {
		if m.app.StatusChecker() == nil {
			return statusUpdateMsg{status: "Unknown", emoji: "⚪"}
		}

		status := m.app.StatusChecker().CheckStatus(context.Background(), windowName)
		var emoji string
		switch status {
		case tmux.Running:
			emoji = "🟢"
		case tmux.Dead:
			emoji = "🔴"
		default:
			emoji = "⚪"
		}

		return statusUpdateMsg{status: status.String(), emoji: emoji}
	}
}

func (m UIModel) startMonitoringCmd(windowName string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg{windowName: windowName}
	})
}

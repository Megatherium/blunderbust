package ui

import (
	"context"
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/data/dolt"
	"github.com/megatherium/blunderbust/internal/domain"
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
		out, err := osexec.Command("bd", "show", ticketID).CombinedOutput()
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

// Agent monitoring commands

func pollAgentStatusCmd(app *App, agentID string, windowName string) tea.Cmd {
	return func() tea.Msg {
		if app.StatusChecker() == nil {
			return AgentStatusMsg{AgentID: agentID, Status: domain.AgentRunning}
		}

		status := app.StatusChecker().CheckStatus(context.Background(), windowName)
		var agentStatus domain.AgentStatus
		switch status {
		case tmux.Running:
			agentStatus = domain.AgentRunning
		case tmux.Dead:
			agentStatus = domain.AgentCompleted
		default:
			agentStatus = domain.AgentRunning
		}

		return AgentStatusMsg{AgentID: agentID, Status: agentStatus}
	}
}

func startAgentMonitoringCmd(agentID string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return agentTickMsg{agentID: agentID}
	})
}

func readAgentOutputCmd(agentID string, capture *tmux.OutputCapture) tea.Cmd {
	return func() tea.Msg {
		if capture == nil {
			return nil
		}

		content, err := capture.ReadOutput()
		if err != nil {
			return nil
		}

		return agentOutputMsg{agentID: agentID, content: string(content)}
	}
}

// Agent clearing commands

func clearAgentCmd(agentID string, capture *tmux.OutputCapture) tea.Cmd {
	return func() tea.Msg {
		// Stop output capture if still running
		if capture != nil {
			capture.Stop(context.Background())
		}

		return AgentClearedMsg{AgentID: agentID}
	}
}

type agentToClear struct {
	id      string
	capture *tmux.OutputCapture
}

func clearAllStoppedAgentsCmd(agents []agentToClear) tea.Cmd {
	return func() tea.Msg {
		var cleared []string
		for _, a := range agents {
			if a.capture != nil {
				a.capture.Stop(context.Background())
			}
			cleared = append(cleared, a.id)
		}

		if len(cleared) > 0 {
			return AllStoppedAgentsClearedMsg{ClearedIDs: cleared}
		}
		return nil
	}
}

// Ticket auto-refresh commands

func checkTicketUpdatesCmd(store data.TicketStore, lastUpdate time.Time) tea.Cmd {
	return func() tea.Msg {
		doltStore, ok := store.(*dolt.Store)
		if !ok {
			return tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
				return ticketUpdateCheckMsg{}
			})
		}

		var dbUpdate time.Time
		err := doltStore.DB().QueryRow("SELECT MAX(updated_at) FROM ready_issues").Scan(&dbUpdate)
		if err != nil {
			return tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
				return ticketUpdateCheckMsg{}
			})
		}

		if !dbUpdate.Equal(lastUpdate) && !dbUpdate.IsZero() {
			return ticketsAutoRefreshedMsg{}
		}

		return tea.Tick(ticketPollingInterval, func(t time.Time) tea.Msg {
			return ticketUpdateCheckMsg{}
		})
	}
}

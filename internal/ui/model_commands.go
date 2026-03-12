package ui

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/megatherium/blunderbust/internal/app"
	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/data"
	"github.com/megatherium/blunderbust/internal/data/dolt"
	"github.com/megatherium/blunderbust/internal/domain"
	"github.com/megatherium/blunderbust/internal/exec/tmux"
	"github.com/megatherium/blunderbust/internal/ui/sidebar"
)

func startServerAndRetryCmd(myApp *app.App, store *dolt.Store) tea.Cmd {
	return func() tea.Msg {
		if myApp == nil || store == nil {
			return errMsg{err: fmt.Errorf("invalid app or store for retry")}
		}

		// Try to start the server
		newStore, err := store.TryStartServer(context.Background())
		if err != nil {
			return errMsg{err: err}
		}

		return serverStartedMsg{store: newStore}
	}
}

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

func discoverWorktreesCmd(myApp *app.App) tea.Cmd {
	return func() tea.Msg {
		projects := myApp.GetProjects()
		if myApp.Opts.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] discoverWorktreesCmd: found %d projects\n", len(projects))
			for i, p := range projects {
				fmt.Fprintf(os.Stderr, "[DEBUG]   project[%d]: dir=%s, name=%s\n", i, p.Dir, p.Name)
			}
		}

		if len(projects) == 0 {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] discoverWorktreesCmd: no projects configured, using fallback\n")
			}
			return worktreesDiscoveredMsg{err: fmt.Errorf("no projects configured")}
		}

		builder := sidebar.NewTreeBuilder()
		nodes, errs := builder.BuildFromProjects(context.Background(), projects)

		if myApp.Opts.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] discoverWorktreesCmd: discovered %d nodes, %d errors\n", len(nodes), len(errs))
		}

		var cmds []tea.Cmd
		if len(errs) > 0 {
			var errMsgs []string
			for _, e := range errs {
				errMsgs = append(errMsgs, e.Error())
			}
			fullErr := strings.Join(errMsgs, "\n")
			cmds = append(cmds, func() tea.Msg {
				return warningMsg{fmt.Errorf("discovery warnings:\n%s", fullErr)}
			})
		}

		cmds = append(cmds, func() tea.Msg {
			return worktreesDiscoveredMsg{nodes: nodes}
		})

		// tea.Sequence to make sure the warnings print after load or whatever?
		// Actually tea.Batch is fine.
		return tea.Batch(cmds...)()
	}
}

func (m UIModel) launchCmd() tea.Cmd {
	return func() tea.Msg {
		workDir := m.selectedWorktree
		if workDir == "" {
			workDir = app.ExtractRepoRoot(m.app.Opts.BeadsDir)
		}

		spec, err := m.app.Renderer.RenderSelection(m.selection, workDir)
		if err != nil {
			return launchResultMsg{
				res: nil,
				err: fmt.Errorf("failed to render launch spec: %w", err),
			}
		}

		spec.LauncherID = m.selection.Ticket.ID

		res, err := m.app.Launcher.Launch(context.Background(), *spec)
		return launchResultMsg{res: res, spec: spec, err: err}
	}
}

func loadRunningAgentsCmd(myApp *app.App) tea.Cmd {
	return func() tea.Msg {
		project := myApp.Project()
		if project == nil || project.Store() == nil {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] loadRunningAgentsCmd: no project or store\n")
			}
			return runningAgentsLoadedMsg{}
		}

		store, ok := project.Store().(*dolt.Store)
		if !ok {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] loadRunningAgentsCmd: store is not dolt.Store\n")
			}
			return runningAgentsLoadedMsg{}
		}

		projectDirs := make([]string, 0, len(myApp.GetProjects()))
		for _, p := range myApp.GetProjects() {
			projectDirs = append(projectDirs, p.Dir)
		}
		if len(projectDirs) == 0 && myApp.ActiveProject != "" {
			projectDirs = append(projectDirs, myApp.ActiveProject)
		}

		if myApp.Opts.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] loadRunningAgentsCmd: querying projectDirs=%v\n", projectDirs)
		}

		if err := store.DeleteStaleRunningAgents(context.Background(), time.Hour); err != nil {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] loadRunningAgentsCmd: DeleteStaleRunningAgents error: %v\n", err)
			}
			return runningAgentsLoadedMsg{err: err}
		}

		agents, err := store.ValidateAndPruneRunningAgents(context.Background(), projectDirs, nil)
		if err != nil {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] loadRunningAgentsCmd: ValidateAndPruneRunningAgents error: %v\n", err)
			}
			return runningAgentsLoadedMsg{err: err}
		}

		if myApp.Opts.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] loadRunningAgentsCmd: loaded %d valid agents\n", len(agents))
			for _, a := range agents {
				fmt.Fprintf(os.Stderr, "[DEBUG]   - %s: PID=%d, harness=%s, binary=%s, worktree=%s\n",
					a.Ticket, a.PID, a.HarnessName, a.HarnessBinary, a.WorktreePath)
			}
		}

		return runningAgentsLoadedMsg{agents: agents}
	}
}

func saveRunningAgentCmd(myApp *app.App, spec *domain.LaunchSpec, result *domain.LaunchResult, worktreePath string) tea.Cmd {
	return func() tea.Msg {
		if myApp == nil || spec == nil || result == nil {
			if myApp != nil && myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: nil check failed (app=%v, spec=%v, result=%v)\n",
					myApp != nil, spec != nil, result != nil)
			}
			return nil
		}

		project := myApp.Project()
		if project == nil || project.Store() == nil {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: no project or store\n")
			}
			return nil
		}
		store, ok := project.Store().(*dolt.Store)
		if !ok {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: store is not dolt.Store\n")
			}
			return nil
		}

		harnessBinary := config.ExtractCommandBinary(spec.RenderedCommand)
		if harnessBinary == "" {
			candidates := config.HarnessBinaryCandidates(spec.Selection.Harness.Name)
			if len(candidates) > 0 {
				harnessBinary = candidates[0]
			}
		}

		projectDir := myApp.ActiveProject
		if projectDir == "" {
			projectDir = app.ExtractRepoRoot(myApp.Opts.BeadsDir)
		}
		if worktreePath == "" {
			worktreePath = projectDir
		}
		if result.PID <= 0 {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: invalid PID %d, not saving\n", result.PID)
			}
			return nil
		}

		if myApp.Opts.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: saving agent\n")
			fmt.Fprintf(os.Stderr, "[DEBUG]   projectDir=%s\n", projectDir)
			fmt.Fprintf(os.Stderr, "[DEBUG]   worktreePath=%s\n", worktreePath)
			fmt.Fprintf(os.Stderr, "[DEBUG]   PID=%d\n", result.PID)
			fmt.Fprintf(os.Stderr, "[DEBUG]   launcherID=%s\n", result.LauncherID)
			fmt.Fprintf(os.Stderr, "[DEBUG]   launcherType=%d\n", result.LauncherType)
			fmt.Fprintf(os.Stderr, "[DEBUG]   ticket=%s\n", spec.Selection.Ticket.ID)
			fmt.Fprintf(os.Stderr, "[DEBUG]   harness=%s\n", spec.Selection.Harness.Name)
			fmt.Fprintf(os.Stderr, "[DEBUG]   harnessBinary=%s\n", harnessBinary)
			fmt.Fprintf(os.Stderr, "[DEBUG]   renderedCommand=%s\n", spec.RenderedCommand)
		}

		err := store.UpsertRunningAgent(context.Background(), domain.PersistedRunningAgent{
			ProjectDir:    projectDir,
			WorktreePath:  worktreePath,
			PID:           result.PID,
			LauncherType:  result.LauncherType,
			LauncherID:    result.LauncherID,
			Ticket:        spec.Selection.Ticket.ID,
			TicketTitle:   spec.Selection.Ticket.Title,
			HarnessName:   spec.Selection.Harness.Name,
			HarnessBinary: harnessBinary,
			Model:         spec.Selection.Model,
			Agent:         spec.Selection.Agent,
		})
		if err != nil {
			if myApp.Opts.Debug {
				fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: UpsertRunningAgent error: %v\n", err)
			}
			return warningMsg{err: fmt.Errorf("failed to persist running agent: %w", err)}
		}

		if myApp.Opts.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] saveRunningAgentCmd: agent saved successfully\n")
		}

		return nil
	}
}

// Agent monitoring commands

func pollAgentStatusCmd(myApp *app.App, agentID, launcherID string) tea.Cmd {
	return func() tea.Msg {
		if myApp.StatusChecker() == nil {
			return AgentStatusMsg{AgentID: agentID, Status: domain.AgentRunning}
		}

		status := myApp.StatusChecker().CheckStatus(context.Background(), launcherID)
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
			_ = capture.Stop(context.Background())
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
		cleared := make([]string, 0, len(agents))
		for _, a := range agents {
			if a.capture != nil {
				_ = a.capture.Stop(context.Background())
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
		dbUpdate, err := store.LatestUpdate(context.Background())
		if err != nil {
			// Check if this is a connection error for server-mode stores
			if doltStore, ok := store.(*dolt.Store); ok &&
				doltStore.CanRetryConnection() &&
				dolt.IsConnectionError(err) {
				return errMsg{err: err}
			}
			// If not retryable or non-server-mode, continue polling silently
			return ticketUpdateCheckNeededMsg{}
		}

		if !dbUpdate.Equal(lastUpdate) && !dbUpdate.IsZero() {
			return ticketsAutoRefreshedMsg{dbUpdatedAt: dbUpdate}
		}

		return ticketUpdateCheckNeededMsg{}
	}
}

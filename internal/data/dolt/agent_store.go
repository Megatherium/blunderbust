package dolt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/megatherium/blunderbust/internal/config"
	"github.com/megatherium/blunderbust/internal/domain"
)

const defaultRunningAgentMaxAge = time.Hour

// ProcessInspector provides process existence and command lookup.
type ProcessInspector interface {
	PIDExists(pid int) bool
	CommandForPID(ctx context.Context, pid int) (string, error)
}

type hostProcessInspector struct{}

func (hostProcessInspector) PIDExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil || err == syscall.EPERM
}

func (hostProcessInspector) CommandForPID(ctx context.Context, pid int) (string, error) {
	cmd := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "command=")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureRunningAgentsTable ensures the running_agents table exists.
func (s *Store) EnsureRunningAgentsTable(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS running_agents (
    id INT PRIMARY KEY AUTO_INCREMENT,
    project_dir VARCHAR(255) NOT NULL,
    worktree_path VARCHAR(255) NOT NULL,
    pid INT NOT NULL,
    tmux_session VARCHAR(100) NOT NULL,
    window_name VARCHAR(100),
    ticket VARCHAR(100),
    ticket_title TEXT,
    harness_name VARCHAR(50) NOT NULL,
    harness_binary VARCHAR(100),
    model VARCHAR(50),
    agent VARCHAR(50),
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uniq_running_agent (project_dir, worktree_path, pid),
    INDEX idx_running_agents_project_dir (project_dir),
    INDEX idx_running_agents_last_seen (last_seen)
)`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to ensure running_agents table: %w", err)
	}
	if err := ensureRunningAgentsTicketTitleColumn(ctx, s); err != nil {
		return err
	}

	return nil
}

// UpsertRunningAgent inserts or updates one running agent row.
func (s *Store) UpsertRunningAgent(ctx context.Context, a domain.PersistedRunningAgent) error {
	if s.closed {
		return fmt.Errorf("store is closed")
	}
	if a.ProjectDir == "" || a.WorktreePath == "" || a.PID <= 0 || a.HarnessName == "" {
		return fmt.Errorf("invalid running agent data")
	}
	if a.TmuxSession == "" {
		a.TmuxSession = "unknown"
	}
	const query = `
INSERT INTO running_agents (
	project_dir, worktree_path, pid, tmux_session, window_name, ticket, ticket_title,
	harness_name, harness_binary, model, agent, started_at, last_seen
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE
	tmux_session = VALUES(tmux_session),
	window_name = VALUES(window_name),
	ticket = VALUES(ticket),
	ticket_title = VALUES(ticket_title),
	harness_name = VALUES(harness_name),
	harness_binary = VALUES(harness_binary),
	model = VALUES(model),
	agent = VALUES(agent),
	last_seen = CURRENT_TIMESTAMP`
	result, err := s.db.ExecContext(ctx, query,
		a.ProjectDir,
		a.WorktreePath,
		a.PID,
		a.TmuxSession,
		a.WindowName,
		a.Ticket,
		a.TicketTitle,
		a.HarnessName,
		a.HarnessBinary,
		a.Model,
		a.Agent,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert running agent: %w", err)
	}
	
	// Debug: check rows affected
	rowsAffected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()
	fmt.Fprintf(os.Stderr, "[DEBUG] UpsertRunningAgent: rowsAffected=%d, lastID=%d, PID=%d, ticket=%s\n",
		rowsAffected, lastID, a.PID, a.Ticket)
	
	return nil
}

// ListRunningAgentsByProjects returns running agents for the given project directories.
func (s *Store) ListRunningAgentsByProjects(ctx context.Context, projectDirs []string) ([]domain.PersistedRunningAgent, error) {
	if s.closed {
		return nil, fmt.Errorf("store is closed")
	}
	if len(projectDirs) == 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: no projectDirs provided\n")
		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: querying for projectDirs=%v\n", projectDirs)

	placeholders := make([]string, 0, len(projectDirs))
	args := make([]any, 0, len(projectDirs))
	for _, dir := range projectDirs {
		placeholders = append(placeholders, "?")
		args = append(args, dir)
	}

	query := fmt.Sprintf(`
SELECT
	id, project_dir, worktree_path, pid, tmux_session, window_name, ticket, ticket_title,
	harness_name, harness_binary, model, agent, started_at, last_seen
FROM running_agents
WHERE project_dir IN (%s)
ORDER BY started_at DESC`, strings.Join(placeholders, ", "))

	fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: SQL query with %d placeholders\n", len(placeholders))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: query error: %v\n", err)
		return nil, fmt.Errorf("failed to query running agents: %w", err)
	}
	defer rows.Close()

	var agents []domain.PersistedRunningAgent
	rowCount := 0
	for rows.Next() {
		rowCount++
		var a domain.PersistedRunningAgent
		if err := rows.Scan(
			&a.ID,
			&a.ProjectDir,
			&a.WorktreePath,
			&a.PID,
			&a.TmuxSession,
			&a.WindowName,
			&a.Ticket,
			&a.TicketTitle,
			&a.HarnessName,
			&a.HarnessBinary,
			&a.Model,
			&a.Agent,
			&a.StartedAt,
			&a.LastSeen,
		); err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: scan error at row %d: %v\n", rowCount, err)
			return nil, fmt.Errorf("failed to scan running agent row: %w", err)
		}
		agents = append(agents, a)
	}
	
	fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: scanned %d rows\n", rowCount)
	
	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: rows.Err(): %v\n", err)
		return nil, fmt.Errorf("error iterating running agent rows: %w", err)
	}
	
	fmt.Fprintf(os.Stderr, "[DEBUG] ListRunningAgentsByProjects: returning %d agents\n", len(agents))
	return agents, nil
}

func ensureRunningAgentsTicketTitleColumn(ctx context.Context, s *Store) error {
	_, err := s.db.ExecContext(ctx, `ALTER TABLE running_agents ADD COLUMN ticket_title TEXT`)
	if err == nil {
		return nil
	}
	// Column already exists on upgraded/newer schemas. Dolt/MySQL variants:
	// - "Duplicate column name 'ticket_title'"
	// - `Column "ticket_title" already exists`
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "duplicate column name") ||
		(strings.Contains(errMsg, "ticket_title") && strings.Contains(errMsg, "already exists")) {
		return nil
	}
	return fmt.Errorf("failed to ensure running_agents.ticket_title column: %w", err)
}

// ValidateAndPruneRunningAgents validates running agents and removes invalid rows.
func (s *Store) ValidateAndPruneRunningAgents(ctx context.Context, projectDirs []string, inspector ProcessInspector) ([]domain.PersistedRunningAgent, error) {
	if inspector == nil {
		inspector = hostProcessInspector{}
	}

	agents, err := s.ListRunningAgentsByProjects(ctx, projectDirs)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] ValidateAndPruneRunningAgents: found %d agents in DB for projectDirs=%v\n", len(agents), projectDirs)

	valid := make([]domain.PersistedRunningAgent, 0, len(agents))
	for _, a := range agents {
		fmt.Fprintf(os.Stderr, "[DEBUG]   validating: %s (PID=%d, harness=%s, binary=%s)\n", 
			a.Ticket, a.PID, a.HarnessName, a.HarnessBinary)
		
		isValid, err := s.validateRunningAgent(ctx, a, inspector)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DEBUG]     validation error: %v\n", err)
			return nil, err
		}
		if !isValid {
			fmt.Fprintf(os.Stderr, "[DEBUG]     -> INVALID (pruned)\n")
			continue
		}

		fmt.Fprintf(os.Stderr, "[DEBUG]     -> VALID\n")
		if err := s.touchRunningAgentByID(ctx, a.ID); err != nil {
			return nil, err
		}
		valid = append(valid, a)
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] ValidateAndPruneRunningAgents: %d/%d agents valid\n", len(valid), len(agents))
	return valid, nil
}

func (s *Store) validateRunningAgent(ctx context.Context, a domain.PersistedRunningAgent, inspector ProcessInspector) (bool, error) {
	if !inspector.PIDExists(a.PID) {
		fmt.Fprintf(os.Stderr, "[DEBUG]     PID %d does not exist\n", a.PID)
		return false, s.deleteRunningAgentByID(ctx, a.ID)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG]     PID %d exists\n", a.PID)

	cmd, err := inspector.CommandForPID(ctx, a.PID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DEBUG]     failed to get command for PID %d: %v\n", a.PID, err)
		return false, s.deleteRunningAgentByID(ctx, a.ID)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG]     command for PID %d: %s\n", a.PID, cmd)

	candidates := config.HarnessBinaryCandidates(a.HarnessName)
	if a.HarnessBinary != "" {
		candidates = append(candidates, a.HarnessBinary)
	}
	
	extractedBinary := config.ExtractCommandBinary(cmd)
	fmt.Fprintf(os.Stderr, "[DEBUG]     harness=%s, candidates=%v\n", a.HarnessName, candidates)
	fmt.Fprintf(os.Stderr, "[DEBUG]     extracted binary from command: %s\n", extractedBinary)
	
	if !config.CommandMatchesAnyBinary(cmd, candidates) {
		fmt.Fprintf(os.Stderr, "[DEBUG]     command does not match any candidate binary\n")
		return false, s.deleteRunningAgentByID(ctx, a.ID)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG]     command matches candidate binary\n")

	return true, nil
}

// DeleteStaleRunningAgents deletes rows older than maxAge by last_seen.
func (s *Store) DeleteStaleRunningAgents(ctx context.Context, maxAge time.Duration) error {
	if maxAge <= 0 {
		maxAge = defaultRunningAgentMaxAge
	}
	cutoff := time.Now().UTC().Add(-maxAge)
	_, err := s.db.ExecContext(ctx, `DELETE FROM running_agents WHERE last_seen < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("failed deleting stale running agents: %w", err)
	}
	return nil
}

func (s *Store) deleteRunningAgentByID(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM running_agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed deleting running agent id=%d: %w", id, err)
	}
	return nil
}

func (s *Store) touchRunningAgentByID(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE running_agents SET last_seen = CURRENT_TIMESTAMP WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed touching running agent id=%d: %w", id, err)
	}
	return nil
}

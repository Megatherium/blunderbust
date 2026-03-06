package dolt

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/megatherium/blunderbust/internal/domain"
)

type fakeInspector struct {
	exists   map[int]bool
	commands map[int]string
	errs     map[int]error
}

func (f fakeInspector) PIDExists(pid int) bool {
	return f.exists[pid]
}

func (f fakeInspector) CommandForPID(_ context.Context, pid int) (string, error) {
	if err, ok := f.errs[pid]; ok {
		return "", err
	}
	return f.commands[pid], nil
}

func TestStore_EnsureRunningAgentsTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS running_agents").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE running_agents ADD COLUMN ticket_title TEXT").
		WillReturnError(errors.New("Error 1060: Duplicate column name 'ticket_title'"))

	if err := store.EnsureRunningAgentsTable(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestStore_EnsureRunningAgentsTable_ColumnAlreadyExistsVariant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS running_agents").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("ALTER TABLE running_agents ADD COLUMN ticket_title TEXT").
		WillReturnError(errors.New(`Error 1105 (HY000): Column "ticket_title" already exists`))

	if err := store.EnsureRunningAgentsTable(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestStore_UpsertRunningAgent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}
	agent := domain.PersistedRunningAgent{
		ProjectDir:    "/repo",
		WorktreePath:  "/repo",
		PID:           1234,
		TmuxSession:   "s0",
		WindowName:    "bb-1",
		Ticket:        "bb-1",
		TicketTitle:   "Test ticket",
		HarnessName:   "kilocode",
		HarnessBinary: "kilo",
		Model:         "m",
		Agent:         "a",
	}

	mock.ExpectExec("INSERT INTO running_agents").
		WithArgs("/repo", "/repo", 1234, "s0", "bb-1", "bb-1", "Test ticket", "kilocode", "kilo", "m", "a").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.UpsertRunningAgent(context.Background(), agent); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ListRunningAgentsByProjects(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id", "project_dir", "worktree_path", "pid", "tmux_session", "window_name", "ticket", "ticket_title",
		"harness_name", "harness_binary", "model", "agent", "started_at", "last_seen",
	}).AddRow(1, "/repo", "/repo", 555, "s0", "bb-1", "bb-1", "Title 1", "kilocode", "kilo", "m", "a", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT
	id, project_dir, worktree_path, pid, tmux_session, window_name, ticket, ticket_title,
	harness_name, harness_binary, model, agent, started_at, last_seen
FROM running_agents
WHERE project_dir IN (?)
ORDER BY started_at DESC`)).
		WithArgs("/repo").
		WillReturnRows(rows)

	got, err := store.ListRunningAgentsByProjects(context.Background(), []string{"/repo"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 1 || got[0].PID != 555 {
		t.Fatalf("unexpected rows: %+v", got)
	}
	if got[0].TicketTitle != "Title 1" {
		t.Fatalf("expected ticket title to be restored, got %q", got[0].TicketTitle)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestStore_ValidateAndPruneRunningAgents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id", "project_dir", "worktree_path", "pid", "tmux_session", "window_name", "ticket", "ticket_title",
		"harness_name", "harness_binary", "model", "agent", "started_at", "last_seen",
	}).
		AddRow(1, "/repo", "/repo", 101, "s0", "bb-1", "bb-1", "Title 1", "kilocode", "kilo", "m", "a", now, now).
		AddRow(2, "/repo", "/repo", 202, "s0", "bb-2", "bb-2", "Title 2", "codex", "codex", "m", "a", now, now).
		AddRow(3, "/repo", "/repo", 303, "s0", "bb-3", "bb-3", "Title 3", "codex", "codex", "m", "a", now, now)

	mock.ExpectQuery("FROM running_agents").
		WithArgs("/repo").
		WillReturnRows(rows)
	mock.ExpectExec("UPDATE running_agents SET last_seen = CURRENT_TIMESTAMP WHERE id = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM running_agents WHERE id = \\?").
		WithArgs(2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM running_agents WHERE id = \\?").
		WithArgs(3).
		WillReturnResult(sqlmock.NewResult(0, 1))

	inspector := fakeInspector{
		exists: map[int]bool{
			101: true,
			202: false,
			303: true,
		},
		commands: map[int]string{
			101: "/usr/local/bin/kilocode --version",
		},
		errs: map[int]error{
			303: errors.New("ps failed"),
		},
	}

	valid, err := store.ValidateAndPruneRunningAgents(context.Background(), []string{"/repo"}, inspector)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(valid) != 1 || valid[0].ID != 1 {
		t.Fatalf("unexpected valid rows: %+v", valid)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

func TestStore_DeleteStaleRunningAgents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	store := &Store{db: db}
	mock.ExpectExec("DELETE FROM running_agents WHERE last_seen < \\?").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.DeleteStaleRunningAgents(context.Background(), time.Hour); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unfulfilled expectations: %v", err)
	}
}

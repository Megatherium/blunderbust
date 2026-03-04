package domain

import "time"

// PersistedRunningAgent represents one row in the running_agents table.
type PersistedRunningAgent struct {
	ID            int
	ProjectDir    string
	WorktreePath  string
	PID           int
	TmuxSession   string
	WindowName    string
	Ticket        string
	HarnessName   string
	HarnessBinary string
	Model         string
	Agent         string
	StartedAt     time.Time
	LastSeen      time.Time
}

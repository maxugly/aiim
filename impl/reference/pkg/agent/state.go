// Package agent: agent lifecycle state machine for AIIM agents.
// Tracks what each agent is doing: IDLE → BUSY → BLOCKED → ERROR.
package agent

import (
	"fmt"
	"sync"
	"time"
)

// State represents an agent's current operational state.
type State int

const (
	StateIdle    State = iota // waiting for work
	StateBusy                  // actively working on a task
	StateBlocked               // waiting on external input (human, other team)
	StateError                 // encountered an error, needs attention
	StateOffline               // agent is unreachable
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateBusy:
		return "busy"
	case StateBlocked:
		return "blocked"
	case StateError:
		return "error"
	case StateOffline:
		return "offline"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// MarshalJSON serializes agent state for .presence.json.
func (s State) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// Agent tracks a single agent's operational state.
type Agent struct {
	mu       sync.Mutex
	ID       string
	State    State
	Task     string
	Since    time.Time
	LastSeen time.Time

	onStateChange func(old, new State)
}

// New creates a new agent in IDLE state.
func New(id string) *Agent {
	now := time.Now()
	return &Agent{
		ID:       id,
		State:    StateIdle,
		Since:    now,
		LastSeen: now,
	}
}

// Transition moves the agent to a new state.
func (a *Agent) Transition(newState State, task string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isValidTransition(a.State, newState) {
		return fmt.Errorf("agent %s: invalid transition %s → %s", a.ID, a.State, newState)
	}

	old := a.State
	a.State = newState
	a.Task = task
	a.Since = time.Now()
	a.LastSeen = time.Now()

	if a.onStateChange != nil {
		a.onStateChange(old, newState)
	}
	return nil
}

// SetTask updates the current task without changing state.
func (a *Agent) SetTask(task string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Task = task
	a.LastSeen = time.Now()
}

// Heartbeat updates LastSeen (called periodically).
func (a *Agent) Heartbeat() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.LastSeen = time.Now()
}

// IsStale returns true if the agent hasn't been seen in the given duration.
func (a *Agent) IsStale(timeout time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return time.Since(a.LastSeen) > timeout
}

// Snapshot returns a copy of the agent's current state for .presence.json.
func (a *Agent) Snapshot() AgentSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return AgentSnapshot{
		ID:       a.ID,
		State:    a.State.String(),
		Task:     a.Task,
		Since:    a.Since.Format(time.RFC3339),
		LastSeen: a.LastSeen.Format(time.RFC3339),
	}
}

// AgentSnapshot is a JSON-safe representation of agent state.
type AgentSnapshot struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	Task     string `json:"task,omitempty"`
	Since    string `json:"since"`
	LastSeen string `json:"last_seen"`
}

// OnStateChange registers a callback for state transitions.
func (a *Agent) OnStateChange(fn func(old, new State)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onStateChange = fn
}

// Valid transitions:
//   IDLE    → BUSY, BLOCKED, ERROR, OFFLINE
//   BUSY    → IDLE, BLOCKED, ERROR
//   BLOCKED → IDLE, BUSY, ERROR
//   ERROR   → IDLE
//   OFFLINE → IDLE
func (a *Agent) isValidTransition(from, to State) bool {
	switch from {
	case StateIdle:
		return to == StateBusy || to == StateBlocked || to == StateError || to == StateOffline
	case StateBusy:
		return to == StateIdle || to == StateBlocked || to == StateError
	case StateBlocked:
		return to == StateIdle || to == StateBusy || to == StateError
	case StateError:
		return to == StateIdle
	case StateOffline:
		return to == StateIdle
	default:
		return false
	}
}

// WorkflowState tracks cross-team workflow states (proposal lifecycle).
type WorkflowState int

const (
	WFOpen      WorkflowState = iota // OPEN — needs attention
	WFVoting                          // VOTING — collecting votes
	WFMerged                          // MERGED — both teams agree
	WFDone                            // DONE — resolved
	WFBlocked                         // BLOCKED — needs human
)

func (w WorkflowState) String() string {
	switch w {
	case WFOpen:
		return "OPEN"
	case WFVoting:
		return "VOTING"
	case WFMerged:
		return "MERGED"
	case WFDone:
		return "DONE"
	case WFBlocked:
		return "BLOCKED"
	default:
		return "UNKNOWN"
	}
}

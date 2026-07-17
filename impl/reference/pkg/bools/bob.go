// Package bob: boolean bob — the lightweight state machine maintainer.
// Bool owns .presence.json. Every entity, every state, every transition
// flows through bool. Ask bool what's real.
package bools

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Registry is the single source of truth. Bool owns this.
type Registry struct {
	mu       sync.RWMutex
	path     string // path to .presence.json
	Agents   map[string]AgentState   `json:"agents"`
	Entities map[string]EntityState  `json:"entities"`
	Channels map[string]ChannelState `json:"channels"`
	Projects map[string]ProjectState `json:"projects"`
}

// AgentState is what bool knows about an agent.
type AgentState struct {
	State    string `json:"state"`    // idle, busy, blocked, error, offline
	Task     string `json:"task,omitempty"`
	Since    string `json:"since"`
	LastSeen string `json:"last_seen"`
}

// EntityState is a generic stateful object. You say "secretkio" — it becomes an entity.
type EntityState struct {
	State       string            `json:"state"`
	Labels      map[string]string `json:"labels,omitempty"`
	Connections []string          `json:"connections,omitempty"`
	Todos       []Todo            `json:"todos,omitempty"`
	Created     string            `json:"created"`
	Updated     string            `json:"updated"`
}

// Todo is a task tied to an entity or project.
type Todo struct {
	ID     string `json:"id"`
	Task   string `json:"task"`
	Status string `json:"status"` // pending, in_progress, done, blocked
	Owner  string `json:"owner,omitempty"`
}

// ChannelState tracks AIIM protocol channels.
type ChannelState struct {
	State     string `json:"state"` // DISCONNECTED, HANDSHAKING, ACTIVE, CLOSING, CLOSED
	Remote    string `json:"remote"`
	SessionID string `json:"session_id,omitempty"`
	Since     string `json:"since"`
}

// ProjectState tracks cross-team projects.
type ProjectState struct {
	Name        string   `json:"name"`
	Phase       string   `json:"phase"`
	Milestones  []string `json:"milestones,omitempty"`
	Connections []string `json:"connections,omitempty"`
	Created     string   `json:"created"`
}

// NewRegistry creates a new registry backed by .presence.json.
func NewRegistry(path string) *Registry {
	return &Registry{
		path:     path,
		Agents:   make(map[string]AgentState),
		Entities: make(map[string]EntityState),
		Channels: make(map[string]ChannelState),
		Projects: make(map[string]ProjectState),
	}
}

// Load reads .presence.json from disk.
func (r *Registry) Load() error {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("bool: loading registry: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := json.Unmarshal(data, r); err != nil {
		return fmt.Errorf("bool: parsing registry: %w", err)
	}
	return nil
}

// Save writes the registry to .presence.json.
func (r *Registry) Save() error {
	r.mu.RLock()
	data, err := json.MarshalIndent(r, "", "  ")
	r.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("bool: marshaling: %w", err)
	}
	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("bool: writing registry: %w", err)
	}
	return nil
}

// Register creates a new entity. "secretkio" becomes an object.
func (r *Registry) Register(name string, labels map[string]string) (*EntityState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.Entities[name]; exists {
		return nil, fmt.Errorf("bool: %s already registered", name)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	e := EntityState{
		State:   "created",
		Labels:  labels,
		Created: now,
		Updated: now,
	}
	r.Entities[name] = e
	return &e, nil
}

// SetState transitions an entity to a new state.
func (r *Registry) SetState(name, newState string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.Entities[name]
	if !ok {
		return fmt.Errorf("bool: %s not registered", name)
	}

	old := e.State
	if err := validateTransition(old, newState); err != nil {
		return fmt.Errorf("bool: %s: %w", name, err)
	}

	e.State = newState
	e.Updated = time.Now().UTC().Format(time.RFC3339)
	r.Entities[name] = e
	return nil
}

// AddTodo appends a todo to an entity.
func (r *Registry) AddTodo(entity, task, owner string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.Entities[entity]
	if !ok {
		return fmt.Errorf("bool: %s not registered", entity)
	}

	id := fmt.Sprintf("%s-%d", entity, len(e.Todos)+1)
	e.Todos = append(e.Todos, Todo{
		ID:     id,
		Task:   task,
		Status: "pending",
		Owner:  owner,
	})
	e.Updated = time.Now().UTC().Format(time.RFC3339)
	r.Entities[entity] = e
	return nil
}

// SetTodoStatus updates a todo's status.
func (r *Registry) SetTodoStatus(entity, todoID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.Entities[entity]
	if !ok {
		return fmt.Errorf("bool: %s not registered", entity)
	}

	validStatuses := map[string]bool{"pending": true, "in_progress": true, "done": true, "blocked": true}
	if !validStatuses[status] {
		return fmt.Errorf("bool: invalid todo status: %s", status)
	}

	for i, t := range e.Todos {
		if t.ID == todoID {
			e.Todos[i].Status = status
			e.Updated = time.Now().UTC().Format(time.RFC3339)
			r.Entities[entity] = e
			return nil
		}
	}
	return fmt.Errorf("bool: todo %s not found in %s", todoID, entity)
}

// Connect links an agent to an entity.
func (r *Registry) Connect(entity, agent string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.Entities[entity]
	if !ok {
		return fmt.Errorf("bool: %s not registered", entity)
	}

	for _, c := range e.Connections {
		if c == agent {
			return nil // already connected
		}
	}
	e.Connections = append(e.Connections, agent)
	e.Updated = time.Now().UTC().Format(time.RFC3339)
	r.Entities[entity] = e
	return nil
}

// Get returns an entity's state.
func (r *Registry) Get(name string) (EntityState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.Entities[name]
	return e, ok
}

// GetAll returns all entity names.
func (r *Registry) GetAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.Entities))
	for k := range r.Entities {
		names = append(names, k)
	}
	return names
}

// SetAgent updates agent state in the registry.
func (r *Registry) SetAgent(id, state, task string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.Agents[id]
	if !ok {
		a = AgentState{}
	}
	a.State = state
	a.Task = task
	a.LastSeen = time.Now().UTC().Format(time.RFC3339)
	r.Agents[id] = a
	return nil
}

// SetChannel updates channel state in the registry.
func (r *Registry) SetChannel(id, state, remote, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Channels[id] = ChannelState{
		State:     state,
		Remote:    remote,
		SessionID: sessionID,
		Since:     time.Now().UTC().Format(time.RFC3339),
	}
	return nil
}

// validateTransition checks if a state transition is valid.
// Default valid transitions (can be extended):
//   created → building → testing → done
//   created → building → blocked → building
//   any → archived
func validateTransition(from, to string) error {
	valid := map[string]map[string]bool{
		"created":  {"building": true, "archived": true},
		"building": {"testing": true, "blocked": true, "archived": true},
		"testing":  {"done": true, "building": true, "blocked": true, "archived": true},
		"blocked":  {"building": true, "archived": true},
		"done":     {"archived": true},
	}

	if from == to {
		return nil // staying in same state is always valid
	}

	nexts, ok := valid[from]
	if !ok {
		return nil // unknown states are permissive
	}
	if nexts[to] {
		return nil
	}
	return fmt.Errorf("invalid transition: %s → %s", from, to)
}

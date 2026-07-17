package bob

import (
	"encoding/json"
	"os"
	"testing"
)

func TestBoolRegisterAndTransition(t *testing.T) {
	r := NewRegistry("")
	r.Register("secretkio", map[string]string{"type": "app", "lang": "go"})

	e, ok := r.Get("secretkio")
	if !ok {
		t.Fatal("secretkio not registered")
	}
	if e.State != "created" {
		t.Errorf("state = %q, want created", e.State)
	}

	// Valid transition
	if err := r.SetState("secretkio", "building"); err != nil {
		t.Errorf("valid transition rejected: %v", err)
	}

	// Invalid transition
	if err := r.SetState("secretkio", "done"); err == nil {
		t.Error("created→done should be invalid (skipped testing)")
	}

	if err := r.SetState("secretkio", "testing"); err != nil {
		t.Errorf("building→testing should be valid: %v", err)
	}
}

func TestBoolTodos(t *testing.T) {
	r := NewRegistry("")
	r.Register("secretkio", nil)

	if err := r.AddTodo("secretkio", "write spec", "bones.714"); err != nil {
		t.Fatalf("AddTodo: %v", err)
	}
	if err := r.AddTodo("secretkio", "implement", "tom.714"); err != nil {
		t.Fatalf("AddTodo: %v", err)
	}

	e, _ := r.Get("secretkio")
	if len(e.Todos) != 2 {
		t.Errorf("expected 2 todos, got %d", len(e.Todos))
	}

	if err := r.SetTodoStatus("secretkio", "secretkio-1", "done"); err != nil {
		t.Errorf("SetTodoStatus: %v", err)
	}

	e, _ = r.Get("secretkio")
	if e.Todos[0].Status != "done" {
		t.Errorf("todo 1 status = %q, want done", e.Todos[0].Status)
	}
}

func TestBoolConnections(t *testing.T) {
	r := NewRegistry("")
	r.Register("secretkio", nil)

	r.Connect("secretkio", "tom.714")
	r.Connect("secretkio", "bones.714")
	r.Connect("secretkio", "tom.714") // duplicate — should be no-op

	e, _ := r.Get("secretkio")
	if len(e.Connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(e.Connections))
	}
}

func TestBoolPersistence(t *testing.T) {
	path := "/tmp/bool-test-presence.json"
	defer os.Remove(path)

	r := NewRegistry(path)
	r.Register("secretkio", nil)
	r.SetState("secretkio", "building")
	r.Save()

	// Load into a new registry
	r2 := NewRegistry(path)
	if err := r2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	e, ok := r2.Get("secretkio")
	if !ok {
		t.Fatal("secretkio lost after reload")
	}
	if e.State != "building" {
		t.Errorf("state = %q, want building", e.State)
	}
}

func TestBoolDualRegister(t *testing.T) {
	r := NewRegistry("")
	r.Register("secretkio", nil)
	_, err := r.Register("secretkio", nil)
	if err == nil {
		t.Error("dual register should error")
	}
}

func TestBoolAgentTracking(t *testing.T) {
	r := NewRegistry("")
	r.SetAgent("tom.714", "busy", "building secretkio")
	r.SetAgent("bones.714", "idle", "")

	if a, ok := r.Agents["tom.714"]; !ok || a.State != "busy" {
		t.Errorf("tom.714 state = %q, want busy", a.State)
	}
}

func TestBoolSnapshotRoundTrip(t *testing.T) {
	r := NewRegistry("")
	r.Register("secretkio", map[string]string{"type": "app"})
	r.AddTodo("secretkio", "build it", "tom.714")
	r.Connect("secretkio", "tom.714")
	r.SetState("secretkio", "building")

	// Marshal/unmarshal round-trip
	data, _ := json.Marshal(r)
	var r2 Registry
	json.Unmarshal(data, &r2)

	e, _ := r2.Get("secretkio")
	if e.State != "building" || len(e.Todos) != 1 || len(e.Connections) != 1 {
		t.Error("round-trip lost data")
	}
}

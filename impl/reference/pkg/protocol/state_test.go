package protocol

import (
	"testing"
	"time"
)

func TestChannelStateTransitions(t *testing.T) {
	ch := NewChannel("agent:test@localhost")

	// Valid transitions
	tests := []struct {
		from, to ChannelState
		valid    bool
	}{
		{StateDisconnected, StateHandshaking, true},
		{StateHandshaking, StateActive, true},
		{StateHandshaking, StateClosing, true},
		{StateHandshaking, StateDisconnected, true},
		{StateActive, StateClosing, true},
		{StateActive, StateDisconnected, true},
		{StateClosing, StateClosed, true},
		{StateClosed, StateDisconnected, true},

		// Invalid transitions
		{StateDisconnected, StateActive, false},
		{StateDisconnected, StateClosing, false},
		{StateDisconnected, StateClosed, false},
		{StateActive, StateHandshaking, false},
		{StateClosed, StateActive, false},
		{StateClosing, StateActive, false},
	}

	for _, tt := range tests {
		t.Run(ch.State.String()+"→"+tt.to.String(), func(t *testing.T) {
			ch.State = tt.from
			err := ch.Transition(tt.to)
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid, got no error")
			}
		})
	}
}

func TestChannelHeartbeat(t *testing.T) {
	ch := NewChannel("agent:test@localhost")
	ch.State = StateActive
	ch.LastFrameAt = time.Now().Add(-61 * time.Second) // idle > 60s

	if !ch.NeedsHeartbeat() {
		t.Error("expected NeedsHeartbeat after 61s idle")
	}

	ch.Touch()
	if ch.NeedsHeartbeat() {
		t.Error("expected no heartbeat needed after Touch")
	}
}

func TestChannelCloseTimeout(t *testing.T) {
	ch := NewChannel("agent:test@localhost")
	ch.State = StateClosing
	ch.LastFrameAt = time.Now().Add(-6 * time.Second) // expired > 5s

	if !ch.IsExpired() {
		t.Error("expected IsExpired after 6s in CLOSING")
	}
}

func TestChannelFullLifecycle(t *testing.T) {
	ch := NewChannel("agent:test@localhost")

	transitions := []ChannelState{
		StateHandshaking,
		StateActive,
		StateClosing,
		StateClosed,
	}

	for _, target := range transitions {
		if err := ch.Transition(target); err != nil {
			t.Fatalf("transition to %s failed: %v", target, err)
		}
		if ch.State != target {
			t.Errorf("state = %s, want %s", ch.State, target)
		}
	}
}

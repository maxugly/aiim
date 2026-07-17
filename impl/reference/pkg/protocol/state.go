// Package protocol: channel state machine per spec §2.
// Implements the full DISCONNECTED → HANDSHAKING → ACTIVE → CLOSING → CLOSED lifecycle.
package protocol

import (
	"fmt"
	"sync"
	"time"
)

// ChannelState represents the protocol-level state of an AIIM channel.
type ChannelState int

const (
	StateDisconnected ChannelState = iota
	StateHandshaking
	StateActive
	StateClosing
	StateClosed
)

func (s ChannelState) String() string {
	switch s {
	case StateDisconnected:
		return "DISCONNECTED"
	case StateHandshaking:
		return "HANDSHAKING"
	case StateActive:
		return "ACTIVE"
	case StateClosing:
		return "CLOSING"
	case StateClosed:
		return "CLOSED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// Channel is a protocol-level channel with state tracking, timeouts, and heartbeat.
type Channel struct {
	mu           sync.Mutex
	State        ChannelState
	RemoteID     string
	SessionID    string
	Version      string
	EstablishedAt time.Time
	LastFrameAt  time.Time

	// Timeouts from spec §4
	helloTimeout   time.Duration
	pingInterval   time.Duration
	pongTimeout    time.Duration
	messageTTL     time.Duration
	goodbyeTimeout time.Duration

	// Callbacks
	onStateChange func(old, new ChannelState)
}

// NewChannel creates a channel in DISCONNECTED state.
func NewChannel(remoteID string) *Channel {
	return &Channel{
		State:          StateDisconnected,
		RemoteID:       remoteID,
		helloTimeout:   30 * time.Second,
		pingInterval:   60 * time.Second,
		pongTimeout:    30 * time.Second,
		messageTTL:     300 * time.Second,
		goodbyeTimeout: 5 * time.Second,
	}
}

// Transition attempts to move the channel to a new state. Returns error if
// the transition is invalid per the protocol state machine.
func (c *Channel) Transition(newState ChannelState) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isValidTransition(c.State, newState) {
		return fmt.Errorf("invalid state transition: %s → %s", c.State, newState)
	}

	old := c.State
	c.State = newState
	c.LastFrameAt = time.Now()

	if newState == StateActive {
		c.EstablishedAt = time.Now()
	}

	if c.onStateChange != nil {
		c.onStateChange(old, newState)
	}

	return nil
}

// OnStateChange registers a callback for state transitions.
func (c *Channel) OnStateChange(fn func(old, new ChannelState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStateChange = fn
}

// Touch updates the last-frame timestamp (resets idle timer).
func (c *Channel) Touch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastFrameAt = time.Now()
}

// IdleDuration returns how long since the last frame was received.
func (c *Channel) IdleDuration() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.LastFrameAt)
}

// NeedsHeartbeat returns true if the channel is ACTIVE and idle longer than PING interval.
func (c *Channel) NeedsHeartbeat() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.State == StateActive && time.Since(c.LastFrameAt) > c.pingInterval
}

// IsExpired returns true if the channel has been in CLOSING state longer than goodbye timeout.
func (c *Channel) IsExpired() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.State != StateClosing {
		return false
	}
	return time.Since(c.LastFrameAt) > c.goodbyeTimeout
}

// isValidTransition validates state transitions against the protocol spec state machine.
//
// Valid transitions:
//   DISCONNECTED → HANDSHAKING  (HELLO sent or received)
//   HANDSHAKING  → ACTIVE       (READY sent or received)
//   HANDSHAKING  → CLOSING      (ACK rejected or timeout)
//   HANDSHAKING  → DISCONNECTED  (transport failure)
//   ACTIVE       → CLOSING      (GOODBYE sent or received)
//   ACTIVE       → DISCONNECTED  (transport failure)
//   CLOSING      → CLOSED       (both GOODBYEs exchanged or timeout)
func (c *Channel) isValidTransition(from, to ChannelState) bool {
	switch from {
	case StateDisconnected:
		return to == StateHandshaking
	case StateHandshaking:
		return to == StateActive || to == StateClosing || to == StateDisconnected
	case StateActive:
		return to == StateClosing || to == StateDisconnected
	case StateClosing:
		return to == StateClosed
	case StateClosed:
		return to == StateDisconnected // allow reconnect
	default:
		return false
	}
}

// Reset returns the channel to DISCONNECTED state (for reconnection).
func (c *Channel) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.State = StateDisconnected
	c.SessionID = ""
}

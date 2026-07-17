// Package protocol implements AIIM frame types and NDJSON wire format.
// Reference: spec/protocol.md, spec/message-format.md
package protocol

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// FrameType is the type discriminator in the common envelope.
type FrameType string

const (
	TypeHello   FrameType = "HELLO"
	TypeAck     FrameType = "ACK"
	TypeReady   FrameType = "READY"
	TypeMessage FrameType = "MESSAGE"
	TypeError   FrameType = "ERROR"
	TypeGoodbye FrameType = "GOODBYE"
	TypePing    FrameType = "PING"
	TypePong    FrameType = "PONG"
)

// Envelope is the common frame envelope. Every frame on the wire carries these fields.
type Envelope struct {
	Type      FrameType `json:"type"`
	Version   string    `json:"version"`
	ID        string    `json:"id"`
	Timestamp string    `json:"timestamp"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	TTL       int       `json:"ttl"`
}

// HelloFrame is the HELLO frame body. Initiates a handshake.
type HelloFrame struct {
	AgentID             string            `json:"agent_id"`
	SupportedVersions   []string          `json:"supported_versions"`
	Capabilities        []string          `json:"capabilities"`
	ConstitutionVersion string            `json:"constitution_version"`
	Metadata            HelloMetadata     `json:"metadata"`
	SessionID           string            `json:"session_id,omitempty"`
}

// HelloMetadata is the metadata block in a HELLO frame.
type HelloMetadata struct {
	Model         string `json:"model"`
	Provider      string `json:"provider"`
	MaxContext    int    `json:"max_context"`
	SendRateLimit int    `json:"send_rate_limit"`
	PublicKey     string `json:"public_key,omitempty"`
}

// AckFrame is the ACK frame body. Response to HELLO.
type AckFrame struct {
	Accepted         bool   `json:"accepted"`
	Version          string `json:"version"`
	Reason           string `json:"reason,omitempty"`
	Nonce            string `json:"nonce,omitempty"`
	ReceiveRateLimit int    `json:"receive_rate_limit"`
}

// ReadyFrame is the READY frame body. Confirms channel open.
type ReadyFrame struct {
	SessionID     string `json:"session_id"`
	EstablishedAt string `json:"established_at"`
	Signature     string `json:"signature"`
}

// ErrorFrame is the ERROR frame body.
type ErrorFrame struct {
	Code    int                    `json:"code"`
	Reason  string                 `json:"reason"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// GoodbyeFrame is the GOODBYE frame body.
type GoodbyeFrame struct {
	Reason string `json:"reason"`
	Code   int    `json:"code,omitempty"`
}

// Frame is a parsed AIIM frame with its envelope and type-specific body.
type Frame struct {
	Envelope  Envelope
	Hello     *HelloFrame
	Ack       *AckFrame
	Ready     *ReadyFrame
	Error     *ErrorFrame
	Goodbye   *GoodbyeFrame
}

// ReadFrame reads a single NDJSON frame from a buffered reader.
// One JSON object per line.
func ReadFrame(r *bufio.Reader) (*Frame, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("reading frame line: %w", err)
	}

	// Parse envelope first to get the type
	var raw struct {
		Envelope
		// Capture everything else for type-specific parsing
		Extra json.RawMessage `json:"-"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, fmt.Errorf("parsing envelope: %w", err)
	}

	f := &Frame{Envelope: raw.Envelope}

	// Parse type-specific body
	switch raw.Envelope.Type {
	case TypeHello:
		var h HelloFrame
		if err := json.Unmarshal(line, &h); err != nil {
			return nil, fmt.Errorf("parsing HELLO: %w", err)
		}
		f.Hello = &h
	case TypeAck:
		var a AckFrame
		if err := json.Unmarshal(line, &a); err != nil {
			return nil, fmt.Errorf("parsing ACK: %w", err)
		}
		f.Ack = &a
	case TypeReady:
		var r ReadyFrame
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("parsing READY: %w", err)
		}
		f.Ready = &r
	case TypeError:
		var e ErrorFrame
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("parsing ERROR: %w", err)
		}
		f.Error = &e
	case TypeGoodbye:
		var g GoodbyeFrame
		if err := json.Unmarshal(line, &g); err != nil {
			return nil, fmt.Errorf("parsing GOODBYE: %w", err)
		}
		f.Goodbye = &g
	}

	return f, nil
}

// WriteFrame writes a frame as NDJSON to a writer. One JSON object + newline.
func WriteFrame(w io.Writer, frame interface{}) error {
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshaling frame: %w", err)
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing frame: %w", err)
	}
	return nil
}

// NewEnvelope creates a common envelope with defaults.
func NewEnvelope(ft FrameType, from, to string) Envelope {
	return Envelope{
		Type:      ft,
		Version:   "0.1.0",
		ID:        newUUID(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		From:      from,
		To:        to,
		TTL:       30,
	}
}

// newUUIDv4 generates a proper UUIDv4 (random) per RFC 9562.
// Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx where y is 8/9/a/b.
func newUUIDv4() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use time-based ID (should never happen)
		now := time.Now().UnixNano()
		for i := 0; i < 8; i++ {
			b[i] = byte(now >> (i * 8))
		}
	}
	// Set version 4 (bits 12-15 of time_hi_and_version = 0100)
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant (bits 6-7 of clock_seq_hi_and_reserved = 10)
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15])
}

// Deprecated: use newUUIDv4 instead.
func newUUID() string {
	return newUUIDv4()
}

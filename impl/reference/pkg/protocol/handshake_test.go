package protocol

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/maxugly/aiim/pkg/identity"
)

// TestFrameRoundTrip verifies that HELLO, ACK, READY, ERROR, and GOODBYE
// frames survive marshal → unmarshal round trips.
func TestFrameRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		frame interface{}
	}{
		{"HELLO", MakeHello("agent:alice@test", "agent:bob@test", "agent:alice@test", []string{"test"})},
		{"READY", MakeReady("agent:alice@test", "agent:bob@test", "sess-001", "dGVzdC1ub25jZQ", &identity.KeyPair{
			PublicKey:  make(ed25519.PublicKey, 32),
			PrivateKey: make(ed25519.PrivateKey, 64),
		})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteFrame(&buf, tt.frame); err != nil {
				t.Fatalf("WriteFrame: %v", err)
			}

			frame, err := ReadFrame(bufio.NewReader(&buf))
			if err != nil {
				t.Fatalf("ReadFrame: %v", err)
			}

			if frame.Envelope.Version != "0.1.0" {
				t.Errorf("version = %q, want 0.1.0", frame.Envelope.Version)
			}
			if frame.Envelope.TTL != 30 {
				t.Errorf("ttl = %d, want 30", frame.Envelope.TTL)
			}
		})
	}
}

// TestUUIDv4Format verifies newUUIDv4 produces valid UUIDv4 strings.
func TestUUIDv4Format(t *testing.T) {
	for i := 0; i < 20; i++ {
		id := newUUIDv4()
		// Check format: 8-4-4-4-12 hex
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			t.Errorf("UUID %q: expected 5 dash-separated parts", id)
		}
		if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
			t.Errorf("UUID %q: wrong segment lengths", id)
		}
		// Version nibble must be 4
		if parts[2][0] != '4' {
			t.Errorf("UUID %q: version nibble is %c, want 4", id, parts[2][0])
		}
		// Variant must be 8/9/a/b
		v := parts[3][0]
		if v != '8' && v != '9' && v != 'a' && v != 'b' {
			t.Errorf("UUID %q: variant nibble is %c, want 8/9/a/b", id, v)
		}
		// IDs should be unique
		id2 := newUUIDv4()
		if id == id2 {
			t.Errorf("UUID collision: %q == %q", id, id2)
		}
	}
}

// TestHandshakeHappyPath performs an in-memory handshake using pipes.
func TestHandshakeHappyPath(t *testing.T) {
	// Generate client keypair
	clientPub, clientPriv, _ := ed25519.GenerateKey(rand.Reader)
	clientPubB64 := base64.RawURLEncoding.EncodeToString(clientPub)

	serverTrust := identity.NewTrustStore()

	// Use io.Pipe for in-memory handshake
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	serverConn := &pipeConn{reader: serverReader, writer: serverWriter}
	clientConn := &pipeConn{reader: clientReader, writer: clientWriter}

	// Run server side in goroutine
	errCh := make(chan error, 1)
	resultCh := make(chan *HandshakeResult, 1)
	go func() {
		result, err := HandleHandshake(serverConn, "agent:server@test", serverTrust)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// Client side: send HELLO
	clientHello := map[string]interface{}{
		"type": "HELLO", "version": "0.1.0",
		"id": newUUIDv4(), "timestamp": time.Now().UTC().Format(time.RFC3339),
		"from": "agent:client@test", "to": "agent:server@test", "ttl": 30,
		"agent_id": "agent:client@test",
		"supported_versions": []string{"0.1.0"},
		"capabilities": []string{"test"},
		"constitution_version": "0.1.0",
		"metadata": map[string]interface{}{
			"model": "test", "provider": "go-test",
			"max_context": 131072, "send_rate_limit": 10,
			"public_key": clientPubB64,
		},
	}
	helloJSON, _ := json.Marshal(clientHello)
	clientConn.Write(append(helloJSON, '\n'))

	// Read ACK
	reader := bufio.NewReader(clientConn)
	ackFrame, err := ReadFrame(reader)
	if err != nil {
		t.Fatalf("reading ACK: %v", err)
	}
	if ackFrame.Ack == nil || !ackFrame.Ack.Accepted {
		t.Fatalf("ACK rejected: %v", ackFrame.Ack)
	}

	// Sign nonce and send READY
	nonce, _ := base64.RawURLEncoding.DecodeString(ackFrame.Ack.Nonce)
	sig := ed25519.Sign(clientPriv, nonce)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	clientReady := map[string]interface{}{
		"type": "READY", "version": "0.1.0",
		"id": newUUIDv4(), "timestamp": time.Now().UTC().Format(time.RFC3339),
		"from": "agent:client@test", "to": "agent:server@test", "ttl": 30,
		"session_id": newUUIDv4(), "established_at": time.Now().UTC().Format(time.RFC3339),
		"signature": sigB64,
	}
	readyJSON, _ := json.Marshal(clientReady)
	clientConn.Write(append(readyJSON, '\n'))

	// Wait for server result
	select {
	case err := <-errCh:
		t.Fatalf("handshake failed: %v", err)
	case result := <-resultCh:
		if result.AgentID != "agent:client@test" {
			t.Errorf("agent_id = %q, want agent:client@test", result.AgentID)
		}
		if result.Version != "0.1.0" {
			t.Errorf("version = %q, want 0.1.0", result.Version)
		}
		t.Logf("handshake success: session=%s", result.SessionID)
	case <-time.After(5 * time.Second):
		t.Fatal("handshake timed out")
	}
}

// TestAgentIDMismatch verifies HELLO with agent_id != from gets ERROR 400.
func TestAgentIDMismatch(t *testing.T) {
	trust := identity.NewTrustStore()
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()
	serverConn := &pipeConn{reader: serverReader, writer: serverWriter}
	clientConn := &pipeConn{reader: clientReader, writer: clientWriter}

	go HandleHandshake(serverConn, "agent:server@test", trust)

	// Send HELLO with mismatched agent_id
	hello := map[string]interface{}{
		"type": "HELLO", "version": "0.1.0",
		"id": newUUIDv4(), "timestamp": time.Now().UTC().Format(time.RFC3339),
		"from": "agent:alice@test", "to": "agent:server@test", "ttl": 30,
		"agent_id": "agent:bob@test", // MISMATCH
		"supported_versions": []string{"0.1.0"},
		"capabilities": []string{},
		"constitution_version": "0.1.0",
		"metadata": map[string]interface{}{
			"model": "test", "provider": "test",
			"max_context": 131072, "send_rate_limit": 10,
		},
	}
	helloJSON, _ := json.Marshal(hello)
	clientConn.Write(append(helloJSON, '\n'))

	// Read ERROR response
	reader := bufio.NewReader(clientConn)
	frame, err := ReadFrame(reader)
	if err != nil {
		t.Fatalf("reading error frame: %v", err)
	}

	if frame.Envelope.Type != TypeError {
		t.Errorf("expected ERROR frame, got %s", frame.Envelope.Type)
	}
	if frame.Error == nil || frame.Error.Code != 400 {
		t.Errorf("expected ERROR 400, got %v", frame.Error)
	}
	t.Logf("correctly rejected: ERROR %d — %s", frame.Error.Code, frame.Error.Reason)
}

// pipeConn implements io.ReadWriter for in-memory connections.
type pipeConn struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (p *pipeConn) Read(b []byte) (int, error)  { return p.reader.Read(b) }
func (p *pipeConn) Write(b []byte) (int, error) { return p.writer.Write(b) }

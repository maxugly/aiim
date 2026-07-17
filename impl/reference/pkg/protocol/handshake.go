// Package protocol: handshake logic for AIIM channel establishment.
// Reference: spec/protocol.md §3
package protocol

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/maxugly/aiim/pkg/identity"
)

// HandshakeResult holds the outcome of a successful handshake.
type HandshakeResult struct {
	SessionID     string
	AgentID       string
	Version       string
	Capabilities  []string
}

// HandleHandshake performs the server side of the AIIM handshake.
// Reads HELLO, validates, sends ACK with nonce, reads READY, verifies signature.
func HandleHandshake(rw io.ReadWriter, serverID string, trust *identity.TrustStore) (*HandshakeResult, error) {
	reader := bufio.NewReader(rw)

	// --- Step 1: Read HELLO ---
	frame, err := ReadFrame(reader)
	if err != nil {
		return nil, fmt.Errorf("reading HELLO: %w", err)
	}
	if frame.Envelope.Type != TypeHello || frame.Hello == nil {
		return nil, fmt.Errorf("expected HELLO, got %s", frame.Envelope.Type)
	}
	hello := frame.Hello

	// Validate agent_id equals from (spec §3.1: MUST reject with ERROR 400)
	if hello.AgentID != frame.Envelope.From {
		writeError(rw, frame.Envelope.From, serverID, 400, "agent_id does not match envelope from field")
		return nil, fmt.Errorf("agent_id mismatch: %s != %s", hello.AgentID, frame.Envelope.From)
	}

	// Validate TTL bounds (spec: min=1, max=86400)
	if frame.Envelope.TTL < 1 || frame.Envelope.TTL > 86400 {
		writeError(rw, frame.Envelope.From, serverID, 400,
			fmt.Sprintf("ttl %d out of range [1, 86400]", frame.Envelope.TTL))
		return nil, fmt.Errorf("ttl out of range: %d", frame.Envelope.TTL)
	}

	// --- Step 2: Version Negotiation ---
	negotiatedVersion, err := negotiateVersion(hello.SupportedVersions, []string{"0.1.0"})
	if err != nil {
		writeAckRejection(rw, frame.Envelope.From, serverID, err.Error())
		return nil, err
	}

	// --- Step 3: Generate Nonce (32 random bytes, RawURLEncoding — no padding) ---
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	nonceB64 := base64.RawURLEncoding.EncodeToString(nonce)

	// --- Step 4: Send ACK ---
	ack := ackWire{
		Type:             TypeAck,
		Version:          "0.1.0",
		ID:               newUUIDv4(),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		From:             serverID,
		To:               frame.Envelope.From,
		TTL:              30,
		Accepted:         true,
		NegotiatedVersion: negotiatedVersion,
		Nonce:            nonceB64,
		ReceiveRateLimit: 10,
	}
	if err := WriteFrame(rw, ack); err != nil {
		return nil, fmt.Errorf("sending ACK: %w", err)
	}

	// --- Step 5: Read READY ---
	frame, err = ReadFrame(reader)
	if err != nil {
		return nil, fmt.Errorf("reading READY: %w", err)
	}
	if frame.Envelope.Type != TypeReady || frame.Ready == nil {
		writeError(rw, frame.Envelope.From, serverID, 400,
			fmt.Sprintf("expected READY, got %s", frame.Envelope.Type))
		return nil, fmt.Errorf("expected READY, got %s", frame.Envelope.Type)
	}
	ready := frame.Ready

	// --- Step 6: Verify Signature ---
	clientPubKey := hello.Metadata.PublicKey
	pubKey, err := identity.DecodePublicKey(clientPubKey)
	if err != nil {
		writeError(rw, frame.Envelope.From, serverID, 400, "invalid or missing public_key in HELLO metadata")
		return nil, fmt.Errorf("decoding client public key: %w", err)
	}

	valid, err := identity.Verify(pubKey, nonce, ready.Signature)
	if err != nil || !valid {
		writeError(rw, frame.Envelope.From, serverID, 401, "signature verification failed")
		return nil, fmt.Errorf("signature verification failed for %s", hello.AgentID)
	}

	// --- Step 7: TOFU — Trust On First Use ---
	if !trust.Record(hello.AgentID, pubKey, hello.ConstitutionVersion) {
		writeError(rw, frame.Envelope.From, serverID, 401,
			"TOFU alert: agent identity key or constitution_version has changed. Operator verification required.")
		return nil, fmt.Errorf("TOFU alert for %s: key or constitution_version changed", hello.AgentID)
	}

	return &HandshakeResult{
		SessionID:    ready.SessionID,
		AgentID:      hello.AgentID,
		Version:      negotiatedVersion,
		Capabilities: hello.Capabilities,
	}, nil
}

// negotiateVersion picks the highest common version respecting client preference order.
func negotiateVersion(clientVersions, serverVersions []string) (string, error) {
	serverSet := make(map[string]bool)
	for _, v := range serverVersions {
		serverSet[v] = true
	}
	for _, v := range clientVersions {
		if serverSet[v] {
			return v, nil
		}
	}
	return "", fmt.Errorf("no common protocol version; client supports %v, server supports %v",
		clientVersions, serverVersions)
}

// ackWire is the flat wire format for ACK frames. Using a flat struct avoids
// JSON tag collision between Envelope.Version and AckFrame's negotiated version.
type ackWire struct {
	Type              FrameType `json:"type"`
	Version           string    `json:"version"`
	ID                string    `json:"id"`
	Timestamp         string    `json:"timestamp"`
	From              string    `json:"from"`
	To                string    `json:"to"`
	TTL               int       `json:"ttl"`
	Accepted          bool      `json:"accepted"`
	NegotiatedVersion string    `json:"negotiated_version"`
	Reason            string    `json:"reason,omitempty"`
	Nonce             string    `json:"nonce,omitempty"`
	ReceiveRateLimit  int       `json:"receive_rate_limit"`
}

// writeAckRejection writes an ACK rejection frame.
func writeAckRejection(w io.Writer, from, to, reason string) {
	ack := ackWire{
		Type:             TypeAck,
		Version:          "0.1.0",
		ID:               newUUIDv4(),
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		From:             to,
		To:               from,
		TTL:              30,
		Accepted:         false,
		NegotiatedVersion: "0.1.0",
		Reason:           reason,
		ReceiveRateLimit: 0,
	}
	WriteFrame(w, ack)
}

// writeError writes an ERROR frame and flushes. Used for fatal protocol errors.
func writeError(w io.Writer, from, to string, code int, reason string) {
	errFrame := struct {
		Envelope
		ErrorFrame
	}{
		Envelope:  NewEnvelope(TypeError, to, from),
		ErrorFrame: ErrorFrame{Code: code, Reason: reason},
	}
	WriteFrame(w, errFrame)
}

// MakeReady creates a READY frame with a signature over the nonce.
func MakeReady(from, to, sessionID, nonceB64 string, keypair *identity.KeyPair) interface{} {
	nonce, err := base64.RawURLEncoding.DecodeString(nonceB64)
	if err != nil {
		nonce = []byte(nonceB64)
	}
	signature := keypair.Sign(nonce)

	return struct {
		Envelope
		ReadyFrame
	}{
		Envelope: NewEnvelope(TypeReady, from, to),
		ReadyFrame: ReadyFrame{
			SessionID:     sessionID,
			EstablishedAt: time.Now().UTC().Format(time.RFC3339),
			Signature:     signature,
		},
	}
}

// MakeHello creates a HELLO frame for initiating a handshake.
func MakeHello(from, to, agentID string, capabilities []string) interface{} {
	return struct {
		Envelope
		HelloFrame
	}{
		Envelope: NewEnvelope(TypeHello, from, to),
		HelloFrame: HelloFrame{
			AgentID:             agentID,
			SupportedVersions:   []string{"0.1.0"},
			Capabilities:        capabilities,
			ConstitutionVersion: "0.1.0",
			Metadata: HelloMetadata{
				Model:         "aiim-reference",
				Provider:      "go-stdlib",
				MaxContext:    131072,
				SendRateLimit: 10,
			},
		},
	}
}

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
// Returns the handshake result or an error.
func HandleHandshake(rw io.ReadWriter, serverID string, keypair *identity.KeyPair, trust *identity.TrustStore) (*HandshakeResult, error) {
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

	// Validate agent_id equals from
	if hello.AgentID != frame.Envelope.From {
		errFrame := buildAckError(frame.Envelope.From, serverID, false, "agent_id does not match envelope from field")
		WriteFrame(rw, errFrame)
		return nil, fmt.Errorf("agent_id mismatch: %s != %s", hello.AgentID, frame.Envelope.From)
	}

	// --- Step 2: Version Negotiation ---
	negotiatedVersion, err := negotiateVersion(hello.SupportedVersions, []string{"0.1.0"})
	if err != nil {
		errFrame := buildAckError(frame.Envelope.From, serverID, false, err.Error())
		WriteFrame(rw, errFrame)
		return nil, err
	}

	// --- Step 3: Generate Nonce ---
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	nonceB64 := base64.URLEncoding.EncodeToString(nonce)

	// --- Step 4: Send ACK ---
	ack := struct {
		Envelope
		AckFrame
	}{
		Envelope: NewEnvelope(TypeAck, serverID, frame.Envelope.From),
		AckFrame: AckFrame{
			Accepted:         true,
			Version:          negotiatedVersion,
			Nonce:            nonceB64,
			ReceiveRateLimit: 10,
		},
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
		// Send ERROR for unexpected frame type
		errFrame := buildError(frame.Envelope.From, serverID, 400, fmt.Sprintf("expected READY, got %s", frame.Envelope.Type))
		WriteFrame(rw, errFrame)
		return nil, fmt.Errorf("expected READY, got %s", frame.Envelope.Type)
	}
	ready := frame.Ready

	// --- Step 6: Verify Signature ---
	// Reference impl shortcut: accept public_key in HELLO metadata until
	// full identity document infrastructure exists (see spec/identity.md).
	// In production, the receiver fetches the identity document to get the key.
	clientPubKey := hello.Metadata.PublicKey
	pubKey, err := identity.DecodePublicKey(clientPubKey)
	if err != nil {
		errFrame := buildError(frame.Envelope.From, serverID, 400, "invalid or missing public_key in HELLO metadata")
		WriteFrame(rw, errFrame)
		return nil, fmt.Errorf("decoding client public key: %w", err)
	}

	// Verify the signature against the CLIENT's public key
	valid, err := identity.Verify(pubKey, nonce, ready.Signature)
	if err != nil || !valid {
		errFrame := buildError(frame.Envelope.From, serverID, 401, "signature verification failed")
		WriteFrame(rw, errFrame)
		return nil, fmt.Errorf("signature verification failed for %s", hello.AgentID)
	}

	// Trust On First Use — record the initiator's key
	trust.Record(hello.AgentID, pubKey)

	return &HandshakeResult{
		SessionID:    ready.SessionID,
		AgentID:      hello.AgentID,
		Version:      negotiatedVersion,
		Capabilities: hello.Capabilities,
	}, nil
}

// negotiateVersion picks the highest common version.
func negotiateVersion(clientVersions, serverVersions []string) (string, error) {
	serverSet := make(map[string]bool)
	for _, v := range serverVersions {
		serverSet[v] = true
	}
	// Client versions are ordered by preference (highest first)
	for _, v := range clientVersions {
		if serverSet[v] {
			return v, nil
		}
	}
	return "", fmt.Errorf("no common protocol version; client supports %v, server supports %v",
		clientVersions, serverVersions)
}

// buildAckError builds a rejection ACK frame.
func buildAckError(from, to string, accepted bool, reason string) interface{} {
	return struct {
		Envelope
		AckFrame
	}{
		Envelope: NewEnvelope(TypeAck, to, from),
		AckFrame: AckFrame{
			Accepted:         accepted,
			Version:          "0.1.0",
			Reason:           reason,
			ReceiveRateLimit: 0,
		},
	}
}

// buildError builds an ERROR frame.
func buildError(from, to string, code int, reason string) interface{} {
	return struct {
		Envelope
		ErrorFrame
	}{
		Envelope: NewEnvelope(TypeError, to, from),
		ErrorFrame: ErrorFrame{
			Code:   code,
			Reason: reason,
		},
	}
}

// MakeReady creates a READY frame with a signature over the nonce.
func MakeReady(from, to, sessionID, nonceB64 string, keypair *identity.KeyPair) interface{} {
	// Decode the nonce to get raw bytes for signing
	nonce, err := base64.URLEncoding.DecodeString(nonceB64)
	if err != nil {
		nonce = []byte(nonceB64) // fallback: sign the encoded string
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

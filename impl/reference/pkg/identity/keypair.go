// Package identity implements AIIM identity primitives: Ed25519 keypairs,
// identity documents, and trust management.
// Reference: spec/identity.md
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// KeyPair is an Ed25519 keypair for an AIIM agent identity.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair creates a new Ed25519 keypair using crypto/rand.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 keypair: %w", err)
	}
	return &KeyPair{PublicKey: pub, PrivateKey: priv}, nil
}

// PublicKeyBase64 returns the public key as a base64url-encoded string.
func (kp *KeyPair) PublicKeyBase64() string {
	return base64.URLEncoding.EncodeToString(kp.PublicKey)
}

// Sign signs data with the private key. Returns base64url-encoded signature.
func (kp *KeyPair) Sign(data []byte) string {
	sig := ed25519.Sign(kp.PrivateKey, data)
	return base64.URLEncoding.EncodeToString(sig)
}

// Verify checks an Ed25519 signature. The signature is base64url-encoded.
func Verify(publicKey ed25519.PublicKey, data []byte, signatureB64 string) (bool, error) {
	sig, err := base64.URLEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("decoding signature: %w", err)
	}
	return ed25519.Verify(publicKey, data, sig), nil
}

// DecodePublicKey decodes a base64url-encoded Ed25519 public key.
func DecodePublicKey(b64 string) (ed25519.PublicKey, error) {
	key, err := base64.URLEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: got %d, want %d", len(key), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(key), nil
}

// TrustStore is a simple in-memory TOFU trust store. Maps agent_id → public key.
type TrustStore struct {
	keys map[string]ed25519.PublicKey
}

// NewTrustStore creates an empty trust store.
func NewTrustStore() *TrustStore {
	return &TrustStore{keys: make(map[string]ed25519.PublicKey)}
}

// Record stores a public key for an agent (Trust On First Use).
// Returns false if the key differs from a previously recorded key (TOFU alert).
func (ts *TrustStore) Record(agentID string, key ed25519.PublicKey) bool {
	existing, ok := ts.keys[agentID]
	if !ok {
		ts.keys[agentID] = key
		return true // first use, trusted
	}
	return existing.Equal(key) // alert if key changed
}

// Get returns the public key for an agent, or nil if unknown.
func (ts *TrustStore) Get(agentID string) ed25519.PublicKey {
	return ts.keys[agentID]
}

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

// trustRecord holds TOFU state for a single agent.
type trustRecord struct {
	PublicKey          ed25519.PublicKey
	ConstitutionVersion string
}

// GenerateKeyPair creates a new Ed25519 keypair using crypto/rand.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating Ed25519 keypair: %w", err)
	}
	return &KeyPair{PublicKey: pub, PrivateKey: priv}, nil
}

// PublicKeyBase64 returns the public key as a base64url-encoded string (no padding).
func (kp *KeyPair) PublicKeyBase64() string {
	return base64.RawURLEncoding.EncodeToString(kp.PublicKey)
}

// Sign signs data with the private key. Returns base64url-encoded signature (no padding).
func (kp *KeyPair) Sign(data []byte) string {
	sig := ed25519.Sign(kp.PrivateKey, data)
	return base64.RawURLEncoding.EncodeToString(sig)
}

// Verify checks an Ed25519 signature. Accepts padded or unpadded base64url.
func Verify(publicKey ed25519.PublicKey, data []byte, signatureB64 string) (bool, error) {
	sig, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		// Fallback: try padded
		sig, err = base64.URLEncoding.DecodeString(signatureB64)
		if err != nil {
			return false, fmt.Errorf("decoding signature: %w", err)
		}
	}
	return ed25519.Verify(publicKey, data, sig), nil
}

// DecodePublicKey decodes a base64url-encoded Ed25519 public key. Accepts padded or unpadded.
func DecodePublicKey(b64 string) (ed25519.PublicKey, error) {
	key, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		key, err = base64.URLEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decoding public key: %w", err)
		}
	}
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key size: got %d, want %d", len(key), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(key), nil
}

// TrustStore is an in-memory TOFU trust store. Maps agent_id → key + constitution_version.
type TrustStore struct {
	records map[string]trustRecord
}

// NewTrustStore creates an empty trust store.
func NewTrustStore() *TrustStore {
	return &TrustStore{records: make(map[string]trustRecord)}
}

// Record stores a public key and constitution_version for an agent (Trust On First Use).
// Returns false if the key or constitution_version differs from a previously recorded
// value, indicating a TOFU alert (spec §5.1 clause 4).
func (ts *TrustStore) Record(agentID string, key ed25519.PublicKey, constitutionVersion string) bool {
	existing, ok := ts.records[agentID]
	if !ok {
		ts.records[agentID] = trustRecord{
			PublicKey:           key,
			ConstitutionVersion: constitutionVersion,
		}
		return true // first use, trusted
	}
	// Alert if either key or constitution_version changed
	if !existing.PublicKey.Equal(key) || existing.ConstitutionVersion != constitutionVersion {
		return false
	}
	return true
}

// Get returns the public key for an agent, or nil if unknown.
func (ts *TrustStore) Get(agentID string) ed25519.PublicKey {
	rec, ok := ts.records[agentID]
	if !ok {
		return nil
	}
	return rec.PublicKey
}

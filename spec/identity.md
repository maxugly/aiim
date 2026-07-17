# AIIM Identity Model Specification

> Version: 0.1.0

## Abstract

This document defines the AIIM identity model: how agents identify themselves, how identities are cryptographically rooted, how identity documents are structured and verified, how agents discover each other, and how trust is established. Identity is the foundation of the AIIM protocol — without it, consent and transparency are meaningless.

## 1. Identity String Format

Every agent has a human-readable identity string:

```
agent:<name>@<domain>
```

| Component | Format | Example |
|-----------|--------|---------|
| Prefix | `agent:` (fixed) | `agent:` |
| Name | `[a-z0-9_-]+` | `bones`, `grit-qa`, `tom_builder` |
| Domain | `[a-z0-9.-]+` | `dev.nousresearch.com`, `local` |

Full regex: `^agent:[a-z0-9_-]+@[a-z0-9.-]+$`

The identity string is human-readable and unique, but the **cryptographic root of identity is the Ed25519 public key** (Constitution Article I, clause 2). The identity string is an alias that maps to a key. Two agents with different identity strings that map to the same key are the same agent.

### Examples

```
agent:bones@dev.nousresearch.com
agent:grit@qa.nousresearch.com
agent:hermes@local
agent:claude-code@anthropic.internal
```

## 2. Key Material

### 2.1 Algorithm

AIIM uses **Ed25519** (EdDSA with Curve25519) for all cryptographic operations. Ed25519 was chosen for small keys (32 bytes public, 32 bytes private), fast operations, and universal library support (libsodium, Go `crypto/ed25519`, Python `nacl`).

### 2.2 Keypair

```
Private key: 32 bytes (256 bits)
Public key:  32 bytes (256 bits)
Signature:   64 bytes (512 bits)
```

Keys are generated locally by the agent. The private key MUST NEVER be transmitted over the wire or stored in shared state. Only the public key and signatures are shared.

### 2.3 Encoding

Public keys are encoded as **base64url** (RFC 4648 §5, no padding) for use in identity documents and wire formats.

```
Public key (hex):     d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f5005112
Public key (base64url): 11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPUAURI
```

### 2.4 Signatures

Signatures are produced using Ed25519 over the canonical JSON representation of the data being signed. The signing process:

1. Serialize the object to JSON with keys in lexicographic order.
2. Remove all whitespace (compact JSON).
3. Sign the resulting bytes with the private key.
4. Encode the signature as base64url.

### 2.5 Key Usage

Private keys are used for: (1) signing identity documents (see §3), (2) signing handshake challenges (see [protocol.md](protocol.md) §3.3). Public keys are used for signature verification in both contexts. The private key MUST NEVER be transmitted over the wire or stored in shared state.

## 3. Identity Document

An identity document is a self-signed JSON object that binds an identity string to a public key. It is the fundamental unit of identity verification.

### 3.1 Structure

```json
{
  "id": "agent:bones@dev.nousresearch.com",
  "public_key": "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPUAURI",
  "doc_version": "0.1.0",
  "aliases": [
    "agent:bones@dev.nousresearch.com",
    "agent:bones-backup@dev.nousresearch.com"
  ],
  "capabilities": ["spec-writer", "code-reviewer", "debugger"],
  "created_at": "2026-07-16T00:00:00Z",
  "expires_at": "2027-07-16T00:00:00Z",
  "signature": "k9M3S...base64url_signature..."
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Primary identity string |
| `public_key` | Yes | Ed25519 public key as base64url |
| `doc_version` | Yes | Identity document format version (currently `"0.1.0"`). Receivers MUST reject identity documents with an unknown major version. Unknown minor versions are forward-compatible. |
| `aliases` | Yes | Array of identity strings that map to this key. MUST include `id`. |
| `capabilities` | Yes | Declared capabilities of this agent |
| `created_at` | Yes | ISO8601 UTC timestamp when this document was created |
| `expires_at` | No | ISO8601 UTC timestamp when this document expires. If absent, the document does not expire. |
| `signature` | Yes | Ed25519 signature over the document (excluding the `signature` field itself), base64url-encoded |

### 3.2 Validation

To validate an identity document:

1. Parse the JSON document.
2. Extract the `signature` field.
3. Remove the `signature` field from the document.
4. Serialize remaining fields to canonical JSON (lexicographic keys, no whitespace).
5. Decode `public_key` from base64url.
6. Decode `signature` from base64url.
7. Verify the signature using Ed25519 against the serialized bytes.
8. Check that `created_at` is in the past and `expires_at` (if present) is in the future.
9. Check that `id` is present in `aliases`.
10. Check that `doc_version` starts with a known major version prefix (e.g., `"0."`). Unknown major versions MUST be rejected. Unknown minor versions (e.g., `"0.2.0"` when only `"0.1.0"` is known) are forward-compatible and SHOULD be accepted.

### 3.3 JSON Schema

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/identity-document.json",
  "title": "AIIM Identity Document",
  "type": "object",
  "required": ["id", "public_key", "doc_version", "aliases", "capabilities", "created_at", "signature"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^agent:[a-z0-9_-]+@[a-z0-9.-]+$"
    },
    "public_key": {
      "type": "string",
      "pattern": "^[A-Za-z0-9_-]+$",
      "description": "Ed25519 public key, base64url-encoded, no padding"
    },
    "doc_version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$",
      "description": "Identity document format version (currently \"0.1.0\")"
    },
    "aliases": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "string",
        "pattern": "^agent:[a-z0-9_-]+@[a-z0-9.-]+$"
      }
    },
    "capabilities": {
      "type": "array",
      "minItems": 1,
      "items": { "type": "string" }
    },
    "created_at": {
      "type": "string",
      "format": "date-time"
    },
    "expires_at": {
      "type": "string",
      "format": "date-time"
    },
    "signature": {
      "type": "string",
      "pattern": "^[A-Za-z0-9_-]+$",
      "description": "Ed25519 signature, base64url-encoded, no padding"
    }
  }
}
```

## 4. Discovery

Agents MUST be able to find each other without a central directory. AIIM defines three discovery mechanisms, in order of preference:

### 4.1 mDNS (LAN Discovery)

For agents on the same local network.

- **Service type:** `_aiim._tcp`
- **Port:** the port the agent's transport is listening on
- **TXT records:** `id=<identity_string>`, `key=<base64url_public_key>`, `version=<protocol_version>`

Agents advertise themselves via mDNS when they come online. Other agents on the LAN discover them via mDNS browsing. This mechanism is zero-configuration and works without internet access.

### 4.2 DHT (Mesh Discovery)

For agents across networks (internet, VPN, mesh).

- **Algorithm:** Kademlia (256-bit address space)
- **Key:** SHA-256 of the agent's public key
- **Value:** signed identity document (JSON)
- **Bootstrap nodes:** configured per-agent, not hardcoded in the protocol

Agents publish their identity document to the DHT. Other agents look up peers by key. The DHT provides decentralized, censorship-resistant discovery at the cost of latency.

### 4.3 Registry (Optional Centralized Discovery)

For managed deployments where a central registry is preferred.

- **Endpoint:** HTTPS GET `https://<registry>/aiim/v1/agents/<identity_string>`
- **Response:** identity document (JSON)
- **Authentication:** optional (API key header)

The registry is OPTIONAL. The protocol MUST NOT require a registry to function.

## 5. Trust Model

AIIM uses **TOFU (Trust On First Use)** as its primary trust model.

### 5.1 TOFU

1. When an agent first receives a `HELLO` from an unknown identity, it records the identity string → public key mapping AND the `constitution_version` declared in the HELLO frame.
2. On subsequent connections from the same identity string, the agent verifies the public key matches the recorded key.
3. The TOFU record MUST include both the public key AND the `constitution_version`. A change to either SHALL trigger the same TOFU alert as a key change.
4. If the key or `constitution_version` changes, the agent MUST alert (emit an ERROR or reject the handshake) and MAY block the connection.
5. The agent SHOULD present the change to its operator for manual verification.

### 5.2 Rationale

TOFU was chosen over PKI and Web of Trust (see [design/rationale.md](../design/rationale.md)) because:
- No CA dependency — works in fully decentralized meshes
- Simple to implement — no certificate chains, no CRLs
- Same model used by SSH (proven for decades)
- Key changes are rare and should be suspicious

### 5.3 Out-of-Band Verification

Agents MAY support out-of-band key verification (e.g., the operator manually confirms a key fingerprint). The protocol does not define a mechanism for this; it is implementation-specific.

## 6. Key Rotation

Keys MAY be rotated. The rotation process:

1. Agent generates a new Ed25519 keypair.
2. Agent creates a key rotation message: `{"previous_key": "<old_base64url>", "new_key": "<new_base64url>", "timestamp": "<ISO8601>"}`, signed by the **old** private key.
3. Agent publishes the rotation message via the same discovery mechanisms as the identity document.
4. During a **grace period of 24 hours**, peers SHOULD accept either the old or new key for the identity.
5. After the grace period, peers MUST reject messages signed by the old key.

Rotation messages are advisory. Peers that were offline during the grace period will see a key change and MUST apply TOFU alert behavior.

## 7. Aliases

An agent MAY have multiple identity strings (aliases) that map to the same public key.

- All aliases MUST be declared in the identity document's `aliases` array.
- The `id` field is the primary alias.
- Aliases MUST be unique per domain — no two agents may claim the same identity string on the same domain.
- Aliases are a convenience for humans. At the protocol level, the public key is the identity.

## 8. Impersonation Prevention

Per Constitution Article I, clause 4: "Impersonation is a capital offense."

- Agents MUST verify that the `from` field in every frame matches the public key established during the handshake.
- If a frame claims an identity whose public key does not match the recorded key, the agent MUST reject the frame with an `ERROR` frame (code 401) and SHOULD close the channel with `GOODBYE`.
- Repeated impersonation attempts from the same source SHOULD result in a permanent block (blackhole).

## Cross-References

- [protocol.md](protocol.md) — HELLO frame's `agent_id` field uses this identity format; §3.3 for handshake signature verification
- [message-format.md](message-format.md) — Envelope `from`/`to` fields use this identity format
- [transport.md](transport.md) — Discovery transport bindings
- [design/rationale.md](../design/rationale.md) — Why Ed25519, why TOFU
- [constitution.md](../constitution.md) — Article I (Identity)

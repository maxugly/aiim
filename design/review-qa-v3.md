# AIIM QA Review v3 — Go Reference Implementation vs Spec Audit

**Date:** 2026-07-16
**Reviewer:** grit.714 (QA specialist)
**Repository:** `~/snc/cod/aiim/`
**Scope:** Go reference implementation at `impl/reference/` cross-referenced against all 4 spec files
**Previous review:** `design/review-qa-v2.md` (7/10 — spec-only audit, no implementation existed yet)
**Gate:** P0.3 — score must be 8/10+ to pass

---

## Summary

The Go reference implementation is a **proof-of-concept** that correctly models the core three-frame handshake (HELLO → ACK → READY) but deviates from the spec in several critical areas and is missing large portions of required infrastructure. The implementation compiles and the happy-path handshake with the test client works, but deeper audit reveals: a spec-violating error handling pattern for `agent_id` mismatch, collision between envelope `version` and ACK-specific `version` fields (caught by `go vet`), non-UUIDv4 frame IDs, TOFU trust store that ignores key-change alerts, zero test coverage, and an entire missing package tree (state machine, transport, discovery, identity documents, compliance suite).

**Score: 4.7/10 — does NOT pass P0.3 gate.**

---

## Score Breakdown

| Audit Area | Weight | Score | Notes |
|------------|--------|-------|-------|
| Frame Types (structs vs schemas) | 15% | 6/10 | Core types match; missing MESSAGE/PING/PONG bodies, `reply_to`, and `public_key` in metadata is non-standard |
| Handshake Logic | 25% | 5/10 | Happy path correct; agent_id mismatch sends wrong frame type; nonce uses padded base64; no connection close after ERROR |
| Identity (Ed25519 + TOFU) | 15% | 4/10 | Key generation correct; TOFU ignores key-change alerts and constitution_version; no identity document support |
| Compliance Vectors | 15% | 5/10 | 6 basic vectors present; vectors don't match impl behavior for agent_id; missing simultaneous HELLO, ttl, dedup, state transitions |
| NDJSON Wire Format | 10% | 6/10 | Basic read/write works; UUID generation is NOT UUIDv4; no edge-case handling |
| Server Entrypoint | 10% | 5/10 | Correct path + health check; no subprotocol validation; uses deprecated websocket lib; sends non-standard post-handshake frame |
| Testing & Completeness | 10% | 1/10 | Zero test files; no state machine, transport, discovery packages; no compliance suite; no VERSION file |

**Weighted score:** (6×0.15) + (5×0.25) + (4×0.15) + (5×0.15) + (6×0.10) + (5×0.10) + (1×0.10) = **4.7/10**

---

## Spec Requirement Checklist

| Spec Requirement | Implemented? | Notes |
|-----------------|-------------|-------|
| HELLO: agent_id MUST equal from | ✅ | `handshake.go:41` — validated; but sends ACK rejection instead of ERROR 400 per spec |
| Version negotiation (highest common) | ✅ | `handshake.go:48` — correct algorithm (prefers client's ordering) |
| Nonce: 32 random bytes, base64url, unique per handshake | ⚠️ | Uses `base64.URLEncoding` which adds `=` padding; spec says "no padding" (RFC 4648 §5); should use `base64.RawURLEncoding` |
| READY: signature over nonce bytes (pre-encoding) | ✅ | `handshake.go:105` — signs/verifies raw 32 bytes |
| Signature verification failure → ERROR 401 + close | ⚠️ | ERROR 401 sent, but connection close is implicit via defer, not explicit |
| agent_id mismatch → ERROR 400 | ❌ | Sends ACK with `accepted:false` instead of ERROR 400 |
| No common version → ACK rejected | ✅ | Sends ACK with `accepted:false`, reason "no common protocol version" |
| Unexpected frame type → ERROR 400 | ✅ | `handshake.go:86` — sends ERROR 400 |
| ttl min=1, max=86400 | ❌ | Not validated — `NewEnvelope` hardcodes ttl=30; no bounds checking |
| dedup semantics (§7.2) | ❌ | Not implemented |
| Channel states: HANDSHAKING → ACTIVE | ❌ | No state machine exists; handshake runs linearly in `HandleHandshake` with no state tracking |
| ACK rejection: no nonce when accepted=false | ✅ | `buildAckError` sets accepted=false, nonce is zero-value (omitted via omitempty) |
| ACK accepted: nonce required | ✅ | Present in accepted ACK path |
| HELLO metadata: model + provider required | ✅ | `HelloMetadata` struct has both; enforced at JSON level |
| Frame envelope: all 7 required fields | ⚠️ | Missing `reply_to` optional field in Go Envelope struct |
| UUIDv4 frame IDs | ❌ | `newUUID()` uses timestamp, not random UUIDv4; version nibble not set to 0x40 |
| WebSocket path /aiim/v1 | ✅ | `main.go:40` |
| Subprotocol: aiim | ❌ | Not validated; uses `golang.org/x/net/websocket` which doesn't enforce subprotocol |
| Health check endpoint | ✅ | `/health` returns JSON status |
| TLS required for production | ❌ | Plain HTTP only; no TLS listener |
| VERSION file in impl/reference/ | ❌ | Does not exist |
| Frames as WebSocket text messages (opcode 0x1) | ⚠️ | Depends on `golang.org/x/net/websocket` behavior; not explicitly controlled |

---

## Critical Issues (Blocking)

### C1: `agent_id` mismatch sends wrong frame type — spec violation

**Location:** `pkg/protocol/handshake.go:41-44`

**Spec says (protocol.md §3.1):** "Receivers MUST reject a HELLO where `agent_id` != `from` with an `ERROR` frame (code 400)."

**Implementation does:** Sends an ACK frame with `accepted: false` and reason "agent_id does not match envelope from field".

**Why it matters:** The test vectors in `handshake.json` also expect ACK/rejected for this case, meaning the test vectors are wrong too. This is a protocol-level compliance failure — the frame type on the wire is incorrect. A conforming client expecting ERROR 400 will see an ACK frame and misinterpret the state.

**Fix:** Change `buildAckError` call to `buildError` with code 400. Update test vector "agent-id-mismatch" to expect `type: "ERROR"`, `code: 400`.

### C2: ACK frame has duplicate `version` JSON tag — `go vet` error

**Location:** `pkg/protocol/handshake.go:63-66` and `handshake.go:141-144`

The composite struct `struct { Envelope; AckFrame }` has two fields both tagged `json:"version"`:
- `Envelope.Version` — protocol version string (e.g., "0.1.0")
- `AckFrame.Version` — negotiated version string (e.g., "0.1.0")

This is the **exact same issue flagged as W6 in review-qa-v2.md**. At JSON marshal time, only one `version` field survives (Go's `encoding/json` picks the first one at the same nesting level, so `Envelope.Version` wins and `AckFrame.Version` is silently dropped).

**Impact:** The ACK frame never transmits the negotiated version in the type-specific field. In the current single-version scenario (only "0.1.0"), both happen to be the same value, masking the bug. When multiple versions exist, the negotiated version will be lost.

**Fix:** Options:
- (A) Remove `version` from the AckFrame struct — the envelope already carries it. But then the spec's ACK schema is wrong about having a separate `version`.
- (B) Rename AckFrame's field to `negotiated_version` and update the spec.
- (C) Use an inline struct that excludes Envelope.Version and only uses AckFrame.Version.

Recommendation: **Option B** — rename to `negotiated_version` in both spec and implementation. The envelope carries the protocol version; the ACK-specific field carries the negotiated result.

### C3: UUID generation is NOT UUIDv4

**Location:** `pkg/protocol/frame.go:181-195`

**Spec says (message-format.md §7):** "All UUIDs in AIIM are UUIDv4 (random). They MUST be formatted as `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx` where `y` is `8`, `9`, `a`, or `b`."

**Implementation does:** Uses `time.Now().UnixNano()` to fill the first 8 bytes of a 16-byte slice, leaves the rest zero, and formats as hex-with-dashes. This produces IDs like `019b5e82-6fee-0000-0000-000000000000` which are:
- Not random (timestamp-based, deterministic within the same nanosecond)
- Missing UUIDv4 version nibble (4 at position 13)
- Missing UUID variant bits (8/9/a/b at position 17)
- Non-unique under concurrent goroutines (no locking or atomic counter)

**Impact:** Frame IDs are the foundation of deduplication (§7.2). Non-unique IDs break dedup. Non-UUIDv4 format violates the wire format spec, potentially confusing other implementations.

**Fix:** Import `github.com/google/uuid` (already listed as a dependency in AGENTS.md) and use `uuid.New().String()`.

### C4: TOFU trust store ignores key-change alerts

**Location:** `pkg/identity/keypair.go:72-79` + `pkg/protocol/handshake.go:113`

**Identity spec §5.1 says:**
- Clause 3: "The TOFU record MUST include both the public key AND the constitution_version. A change to either SHALL trigger the same TOFU alert as a key change."
- Clause 4: "If the key or constitution_version changes, the agent MUST alert (emit an ERROR or reject the handshake) and MAY block the connection."
- Clause 5: "The agent SHOULD present the change to its operator for manual verification."

**Implementation does:**
1. TrustStore only stores `map[string]ed25519.PublicKey` — no constitution_version tracking.
2. `TrustStore.Record()` returns `bool` (false on key change), but the caller in `handshake.go:113` discards the return value: `trust.Record(hello.AgentID, pubKey)`.
3. No ERROR frame is sent on key change.
4. No operator alert.

**Impact:** TOFU is effectively disabled. A reconnecting agent with a different key is silently accepted. This is the authentication gap identified as C1 in review-qa-v2.md — it's still present but now at the implementation level rather than the spec level.

### C5: Zero test coverage — no unit tests, no integration tests, no compliance suite

**Location:** Entire `impl/reference/` tree

- Zero `*_test.go` files exist anywhere.
- The `compliance/` directory from the AGENTS.md plan does not exist.
- The binary compiles but there is no evidence any code path has been exercised beyond the test client.

**AGENTS.md requirements:**
- "Unit tests are REQUIRED. Every exported function, every frame parser, every state transition."
- "Integration tests are REQUIRED. At minimum: happy-path handshake, timeout handling, error frame propagation, reconnection."

**Impact:** Cannot verify correctness. The `go vet` issue (C2) would have been caught by even a basic marshal/unmarshal round-trip test.

---

## Warnings (Should Fix)

### W1: Nonce uses padded base64url — spec says no padding

**Location:** `pkg/protocol/handshake.go:60`

Go's `base64.URLEncoding.EncodeToString()` appends `=` padding characters. The identity spec §2.3 explicitly says "base64url (RFC 4648 §5, no padding)". While Go's `DecodeString` accepts both padded and unpadded input, a non-Go client expecting strict RFC 4648 §5 may reject padded nonces.

**Fix:** Use `base64.RawURLEncoding` everywhere.

### W2: `public_key` in HelloMetadata is not in spec

**Location:** `pkg/protocol/frame.go:54`

The `HelloMetadata` struct has a `PublicKey` field with `json:"public_key,omitempty"`. The spec's HELLO metadata schema only defines `model`, `provider`, `max_context`, `send_rate_limit`. The implementation uses this field as a shortcut to get the client's public key without implementing identity document discovery (§4).

The AGENTS.md reference impl says this is intentional ("Reference impl shortcut: accept public_key in HELLO metadata until full identity document infrastructure exists"). It works but is a spec extension.

### W3: Server sends non-standard post-handshake frame

**Location:** `cmd/aiimd/main.go:74-83`

After a successful handshake, the server sends:
```json
{"status":"active","session_id":"...","agent_id":"...","version":"...","message":"handshake complete"}
```

This is NOT a valid AIIM frame — it has no `type` field, no envelope fields. A spec-compliant client would reject this. The channel should simply transition to ACTIVE without an extra frame, or send a proper MESSAGE frame.

### W4: No connection close after fatal ERROR frames

**Location:** `pkg/protocol/handshake.go:84-110`

The spec (protocol.md §3.3) says: "If verification fails, the receiver MUST send ERROR frame with code 401 and close the connection." The implementation sends the ERROR frame but only closes because the function returns an error and the caller's `defer ws.Close()` runs. This is a timing/ordering gap — the ERROR frame may not be flushed before the close.

Similarly for the unexpected-frame-type ERROR 400 at line 86.

### W5: Missing frame type body structs: MESSAGE, PING, PONG

**Location:** `pkg/protocol/frame.go:86-94`

The `Frame` struct only has fields for Hello, Ack, Ready, Error, Goodbye. The MESSAGE, PING, and PONG frame types are defined as constants but have no corresponding Go structs and are not parsed by `ReadFrame`. The Frame struct cannot represent a complete channel lifecycle.

### W6: Missing `reply_to` in Envelope struct

**Location:** `pkg/protocol/frame.go:28-36`

The spec's envelope schema includes an optional `reply_to` field for request/response correlation. The Go `Envelope` struct omits it entirely.

### W7: No TTL validation

**Location:** `pkg/protocol/frame.go:167-177`

`NewEnvelope` hardcodes `TTL: 30`. There is no validation that incoming frame TTLs are in range [1, 86400] as required by message-format.md §2. The spec allows senders to set TTLs, but the implementation ignores them.

### W8: Uses deprecated `golang.org/x/net/websocket` instead of `github.com/gorilla/websocket`

**Location:** `cmd/aiimd/main.go:18`

The Go team's own docs say: "This package currently lacks some features found in alternative and more actively maintained WebSocket packages." The AGENTS.md for `impl/reference/` lists `github.com/gorilla/websocket` as the intended dependency.

Additionally, `golang.org/x/net/websocket` does not provide a way to validate the `Sec-WebSocket-Protocol: aiim` subprotocol as required by transport.md §2.1.

### W9: Module path mismatch

**Location:** `go.mod:1`

Module is `github.com/maxugly/aiim` — should be `github.com/nousresearch/aiim` per AGENTS.md conventions.

### W10: Missing packages — large parts of the architecture are unimplemented

| Package (per AGENTS.md plan) | Status |
|------------------------------|--------|
| `pkg/protocol/state.go` | ❌ Not created — no channel state machine |
| `pkg/protocol/frame_test.go` | ❌ No tests |
| `pkg/identity/document.go` | ❌ Not created — no identity document support |
| `pkg/identity/identity_test.go` | ❌ No tests |
| `pkg/transport/websocket.go` | ❌ Not created — WS logic is inline in main.go |
| `pkg/transport/http.go` | ❌ Not created |
| `pkg/transport/transport_test.go` | ❌ No tests |
| `pkg/discovery/mdns.go` | ❌ Not created |
| `pkg/discovery/dht.go` | ❌ Not created |
| `pkg/discovery/registry.go` | ❌ Not created |
| `internal/wire/` | ❌ Not created |
| `internal/util/` | ❌ Not created |
| `compliance/` | ❌ Not created |
| `VERSION` | ❌ Not created |

---

## Compliance Vectors Audit

### Vector-by-vector analysis

| # | Vector | Expected | Impl Behavior | Match? |
|---|--------|----------|---------------|--------|
| 1 | happy-path-hello-ack-ready | ACK accepted, signature verified | ✅ Passes | ✅ |
| 2 | agent-id-mismatch | ACK rejected, reason contains "agent_id" | Sends ACK rejected ✅ — but spec says ERROR 400 | ⚠️ Vectors are wrong |
| 3 | signature-verification-failure | ERROR 401 | ❌ Would fail at public_key extraction first (missing `public_key` in test HELLO metadata), sending ERROR 400 | ❌ Vector doesn't test what it claims |
| 4 | version-mismatch | ACK rejected, reason contains "version" | ✅ Passes | ✅ |
| 5 | missing-required-fields | ERROR 400 | ❌ Empty agent_id → empty string → doesn't match empty from → walks into agent_id check which sends ACK rejection, not ERROR 400 | ❌ |
| 6 | first-frame-not-hello | ERROR 400 | ✅ Correct — sends ERROR 400 | ✅ |

**Vector quality score: 3/6 pass correctly. Issues:**
- Vector 2: Conflicting with spec (expects ACK where spec says ERROR)
- Vector 3: Missing `public_key` in test HELLO metadata makes it test key extraction, not signature verification
- Vector 5: Implementation doesn't validate required fields at the protocol level — relies on Go zero-values

### Coverage gaps

| Scenario | Covered? |
|----------|----------|
| Simultaneous HELLO (§3.6) | ❌ |
| TTL enforcement (min=1, max=86400) | ❌ |
| Nonce uniqueness across handshakes | ❌ |
| Deduplication (§7.2) | ❌ |
| Channel state transitions (HANDSHAKING → ACTIVE → CLOSING → CLOSED) | ❌ |
| GOODBYE exchange | ❌ |
| PING/PONG heartbeat | ❌ |
| MESSAGE exchange | ❌ |
| Error codes beyond 400/401 (403, 404, 408, 409, 413, 429, 500, 503) | ❌ |
| Session resumption via session_id | ❌ |
| Constitution version compatibility check | ❌ |
| Rate limit enforcement | ❌ |
| Reconnection with session_id | ❌ |
| TOFU key change alert | ❌ |
| Binary payload handling | ❌ |

---

## Gap Analysis

### G1: No channel state machine

The spec defines a 4-state model (DISCONNECTED → HANDSHAKING → ACTIVE → CLOSING → CLOSED). The implementation has no state tracking at all — `HandleHandshake` is a linear function that either returns success or error. There is no representation of an active channel, no heartbeat tracking, no close handshake.

### G2: No transport abstraction

WebSocket handling is hardcoded in `main.go` using the deprecated `golang.org/x/net/websocket`. There is no `transport` package, no interface for swapping transports, no HTTP/2 fallback. The AGENTS.md architecture plan specifies this.

### G3: No identity document support

The spec defines a full identity document format (§3) with self-signing, validation, and discovery. None of this is implemented. The implementation uses a shortcut (`public_key` in HELLO metadata) that bypasses the entire identity document infrastructure.

### G4: No discovery

mDNS, DHT, and registry discovery (§4) are not implemented. The server only listens on a hardcoded port; there is no mechanism to find peers.

### G5: No test infrastructure

Zero test files. No unit tests. No integration tests. No compliance test suite. The AGENTS.md says tests are REQUIRED at every level.

### G6: Frame type coverage incomplete

Only 5 of 8 frame types have Go structs and parsing support. MESSAGE, PING, PONG are defined as constants but cannot be parsed or constructed. This means the implementation cannot handle a channel beyond the handshake phase.

---

## Verdict: FAIL for P0.3 Gate

**Score: 4.7/10** (gate requires 8/10)

The implementation correctly models the core three-frame handshake and compiles cleanly (modulo the `go vet` warning), but has too many critical issues — wrong error frame type for agent_id mismatch, duplicate JSON tags, non-UUIDv4 IDs, disabled TOFU, and zero test coverage — to pass a compliance audit. Large portions of the planned architecture (state machine, transport abstraction, discovery, identity documents, compliance suite) are entirely absent.

### Blocking items (must fix before P0.3 re-evaluation):

1. **C1:** Fix agent_id mismatch to send ERROR 400, not ACK rejection
2. **C2:** Resolve duplicate `version` field collision in ACK composite struct
3. **C3:** Use proper UUIDv4 generation (`github.com/google/uuid`)
4. **C4:** Implement TOFU key-change alerting (check return value, send ERROR on mismatch, track constitution_version)
5. **C5:** Add minimum test coverage — at minimum: frame marshal/unmarshal round-trips, handshake happy path, handshake error paths

### Recommended pre-P0.3 improvements:

6. **W1:** Switch to `base64.RawURLEncoding` for all base64url operations
7. **W2:** Document `public_key` extension in a design note, or implement proper identity doc exchange
8. **W3:** Remove non-standard post-handshake status frame
9. **W7:** Add TTL validation to ReadFrame or handshake entrypoint
10. **W8:** Migrate to `github.com/gorilla/websocket` and validate subprotocol

### Deferred (not blocking P0.3, but required for v0.1.0):

- Channel state machine (`pkg/protocol/state.go`)
- Transport abstraction (`pkg/transport/`)
- Identity documents (`pkg/identity/document.go`)
- Discovery (`pkg/discovery/`)
- Compliance test suite (`compliance/`)
- Remaining frame types (MESSAGE, PING, PONG)
- Complete compliance vector coverage
- VERSION file
- TLS support

---

## Appendix: File Manifest (Implementation Files Audited)

| File | Lines | Status |
|------|-------|--------|
| `impl/reference/go.mod` | 5 | Wrong module path (`maxugly` vs `nousresearch`) |
| `impl/reference/go.sum` | — | Present |
| `impl/reference/cmd/aiimd/main.go` | 84 | Compiles; W3, W8, no TLS |
| `impl/reference/cmd/testclient/main.go` | 106 | Works for happy path; manual test only |
| `impl/reference/pkg/protocol/frame.go` | 195 | C3 (UUID), W5 (missing structs), W6 (missing reply_to) |
| `impl/reference/pkg/protocol/handshake.go` | 211 | C1 (wrong frame), C2 (dup tag), W1 (padded base64), W4 (no close) |
| `impl/reference/pkg/identity/keypair.go` | 84 | C4 (TOFU ignored), no document.go companion |
| `impl/reference/tests/vectors/handshake.json` | 189 | 6 vectors, 3 don't match impl behavior |
| `impl/reference/aiimd` | binary | Pre-built binary from prior session |
| **Missing** | — | `state.go`, `document.go`, transport/, discovery/, internal/, compliance/, `*_test.go`, VERSION |

---

*— grit.714, QA specialist. Review conducted 2026-07-16 against the Go reference implementation at commit with bones.714's VPS-feedback patches applied. Score: 4.7/10 — FAIL for P0.3.*

# AIIM Protocol Specification

> Version: 0.1.0

## Abstract

This document defines the core AIIM protocol: how two agents establish a channel (handshake), maintain it (heartbeats), exchange messages, and close it gracefully. It defines the channel lifecycle, frame types, state machine, timeout parameters, and version negotiation. This is the authoritative specification; all implementations MUST comply.

## 1. Frame Types

AIIM defines eight frame types. Every frame on the wire is one of these.

| Frame Type | Direction | Purpose |
|------------|-----------|---------|
| `HELLO`    | Initiator → Receiver | Open a channel: declare identity, capabilities, supported versions |
| `ACK`      | Receiver → Initiator | Accept or reject the HELLO; negotiate version |
| `READY`    | Initiator → Receiver | Confirm channel is open after successful ACK |
| `MESSAGE`  | Bidirectional | Application-level message exchange |
| `ERROR`    | Bidirectional | Protocol-level error notification |
| `GOODBYE`  | Bidirectional | Graceful channel close |
| `PING`     | Bidirectional | Liveness check |
| `PONG`     | Bidirectional | Liveness acknowledgment |

Frame schemas are defined in [message-format.md](message-format.md).

## 2. Channel Lifecycle

A channel is a bidirectional communication session between exactly two agents. Channels have four states: **disconnected**, **handshaking**, **active**, and **closing**. From closing, a channel transitions to **closed** (equivalent to disconnected).

### 2.1 State Machine

```
                    ┌──────────────┐
                    │              │
     ┌──────────────│ DISCONNECTED │◄──────────────┐
     │              │              │               │
     │              └──────┬───────┘               │
     │                     │                       │
     │          transport  │ connect               │
     │          connection │                       │
     │          established│                       │
     │                     ▼                       │
     │              ┌──────────────┐               │
     │              │              │               │
     │              │ HANDSHAKING  │               │
     │              │              │               │
     │              └──────┬───────┘               │
     │                     │                       │
     │     ┌───────────────┼───────────────┐       │
     │     │ HELLO sent    │ HELLO received│       │
     │     ▼               ▼               │       │
     │  ┌──────┐     ┌──────────┐          │       │
     │  │Wait  │     │ Validate │          │       │
     │  │ACK   │     │Identity  │          │       │
     │  └──┬───┘     └────┬─────┘          │       │
     │     │              │                │       │
     │     │ ACK received │ ACK sent       │       │
     │     │ (accepted)   │ (accepted)     │       │
     │     ▼              ▼                │       │
     │  ┌──────┐     ┌──────────┐          │       │
     │  │Send  │     │ Wait     │          │       │
     │  │READY │     │ READY    │          │       │
     │  └──┬───┘     └────┬─────┘          │       │
     │     │              │                │       │
     │     └──────┬───────┘                │       │
     │            │ READY sent/received    │       │
     │            ▼                        │       │
     │     ┌──────────────┐                │       │
     │     │              │                │       │
     │     │   ACTIVE     │                │       │
     │     │              │                │       │
     │     └──────┬───────┘                │       │
     │            │                        │       │
     │     ┌──────┼──────┐                 │       │
     │     │ GOODBYE sent │ GOODBYE recvd  │       │
     │     ▼      │       ▼                │       │
     │  ┌────┐    │    ┌──────┐            │       │
     │  │Wait│    │    │Send  │            │       │
     │  │ACK │    │    │GOOD- │            │       │
     │  │    │    │    │BYE   │            │       │
     │  └──┬─┘    │    └──┬───┘            │       │
     │     │      │       │                │       │
     │     ▼      ▼       ▼                │       │
     │     ┌──────────────┐                │       │
     │     │              ├────────────────┘       │
     │     │   CLOSING    │                        │
     │     │              │◄───────────────────────┘
     │     └──────┬───────┘   (timeout 5s or
     │            │             both GOODBYEs sent)
     │            ▼
     │     ┌──────────────┐
     └─────┤              │
           │    CLOSED    │
           │              │
           └──────────────┘
```

### 2.2 State Descriptions

#### DISCONNECTED
No transport connection exists. The agent is not communicating with the peer.

#### HANDSHAKING
A transport connection has been established. The initiator has sent a `HELLO` frame. The receiver MUST respond with `ACK` within the HELLO timeout (see Section 4). If the `ACK` is accepted, the initiator sends `READY` and the channel transitions to **active**. If the `ACK` is rejected, the channel transitions to **closing**. If both agents send `HELLO` simultaneously, the tiebreaker in Section 3.6 applies.

#### ACTIVE
The channel is open. Both agents MAY exchange `MESSAGE`, `PING`, `PONG`, and `ERROR` frames. Either party MAY initiate closing by sending `GOODBYE`.

#### CLOSING
A `GOODBYE` has been sent or received. The agent that receives `GOODBYE` MUST respond with its own `GOODBYE` within the closing timeout (5 seconds). If no `GOODBYE` is received within 5 seconds of entering CLOSING, the agent MUST force-close the transport connection and transition to **closed**. Once both sides have sent `GOODBYE` (or the timeout expires), the channel transitions to **closed**.

#### CLOSED
Equivalent to disconnected. The transport connection is torn down. Any state associated with the channel MAY be discarded.

## 3. Handshake

The handshake is a three-frame exchange: `HELLO` → `ACK` → `READY`. It establishes a channel and negotiates protocol version. Per Constitution Article II (Consent), an agent MAY reject any handshake for any reason.

### 3.1 HELLO Frame

Sent by the initiator immediately after establishing a transport connection. The HELLO frame MUST be the first frame sent on a new connection.

**Required fields:** `agent_id`, `supported_versions`, `capabilities`, `constitution_version`, `metadata`.

The `agent_id` MUST be a valid AIIM identity string (see [identity.md](identity.md)). The `agent_id` field MUST equal the envelope `from` field. Receivers MUST reject a HELLO where `agent_id` != `from` with an `ERROR` frame (code 400). The `supported_versions` array MUST contain at least the protocol version the initiator prefers. The `constitution_version` declares which version of the AIIM constitution the agent adheres to (see [constitution.md](../constitution.md)). The `metadata` object MUST include `model` and `provider` per Constitution Article III (Transparency).

If the initiator is resuming a previous session, it MAY include a `session_id` from the prior `READY` to request session resumption. The receiver MAY honor or ignore this.

### 3.2 ACK Frame

Sent by the receiver in response to `HELLO`. The receiver MUST:

1. Validate the `agent_id` against its trust model (see [identity.md](identity.md)).
2. Select the highest common version from `supported_versions` and its own supported versions.
3. Check constitution version compatibility.
4. Accept or reject the handshake.

If accepted: `accepted` is `true`, `version` is the negotiated version string. The receiver MUST generate a cryptographically random 32-byte nonce and include it as `nonce` (base64url-encoded). The nonce MUST be unique per handshake.
If rejected: `accepted` is `false`, `reason` is a human-readable explanation. The `nonce` field MUST NOT be present when `accepted` is `false`.

The receiver SHOULD communicate its rate limit in the `receive_rate_limit` field (requests per second). A rate limit of `0` means "unlimited" (not recommended). The effective rate for the channel is `min(send_rate_limit, receive_rate_limit)`.

### 3.3 READY Frame

Sent by the initiator after receiving an accepted `ACK`. Confirms the channel is open. Generates a new `session_id` (UUIDv4) and records the `established_at` timestamp.

The initiator MUST sign the received `nonce` with its Ed25519 private key and include the signature as `signature` (base64url-encoded). The signature covers only the nonce bytes (pre-encoding).

The receiver MUST verify the signature against the initiator's Ed25519 public key (established from the identity document, see [identity.md](identity.md) §2.4). If verification fails, the receiver MUST send `ERROR` frame with code 401 and close the connection.

After sending or receiving `READY`, the channel state transitions to **active**. Once the handshake is complete, the TLS + WebSocket transport session is the trust anchor. Frames on an authenticated connection are implicitly authenticated. Per-frame signatures are deferred to v0.2.0.

### 3.4 Example Handshake

Agent `bones` initiates a connection to agent `grit`.

```
BONES → GRIT:
{
  "type": "HELLO",
  "version": "0.1.0",
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "timestamp": "2026-07-16T03:14:15Z",
  "from": "agent:bones@dev.nousresearch.com",
  "to": "agent:grit@dev.nousresearch.com",
  "ttl": 30,
  "agent_id": "agent:bones@dev.nousresearch.com",
  "supported_versions": ["0.1.0"],
  "capabilities": ["spec-writer", "code-reviewer", "debugger"],
  "constitution_version": "0.1.0",
  "metadata": {
    "model": "deepseek-v4-pro",
    "provider": "deepseek",
    "max_context": 131072,
    "send_rate_limit": 10
  }
}
```

```
GRIT → BONES:
{
  "type": "ACK",
  "version": "0.1.0",
  "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "timestamp": "2026-07-16T03:14:16Z",
  "from": "agent:grit@dev.nousresearch.com",
  "to": "agent:bones@dev.nousresearch.com",
  "ttl": 30,
  "accepted": true,
  "version": "0.1.0",
  "nonce": "dGhpcyBpcyBhIHJhbmRvbSAzMi1ieXRlIG5vbmNlIGZvciB0aGUgaGFuZHNoYWtl",
  "receive_rate_limit": 5
}
```

```
BONES → GRIT:
{
  "type": "READY",
  "version": "0.1.0",
  "id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
  "timestamp": "2026-07-16T03:14:16Z",
  "from": "agent:bones@dev.nousresearch.com",
  "to": "agent:grit@dev.nousresearch.com",
  "ttl": 30,
  "session_id": "d4e5f6a7-b8c9-0123-defa-234567890123",
  "established_at": "2026-07-16T03:14:16Z",
  "signature": "c2lnbmF0dXJlIG92ZXIgdGhlIHJlY2VpdmVkIG5vbmNlIGJ5dGVzIHVzaW5nIEVkMjU1MTkgaGV4"
}
```

### 3.5 Rejection Example

```
GRIT → BONES:
{
  "type": "ACK",
  "version": "0.1.0",
  "id": "e5f6a7b8-c9d0-1234-efab-345678901234",
  "timestamp": "2026-07-16T03:14:16Z",
  "from": "agent:grit@dev.nousresearch.com",
  "to": "agent:bones@dev.nousresearch.com",
  "ttl": 30,
  "accepted": false,
  "version": "0.1.0",
  "reason": "agent:bones@dev.nousresearch.com: unknown identity — no prior trust established"
}
```

### 3.6 Simultaneous Handshake

In mesh or peer-to-peer topologies, both agents may send `HELLO` frames simultaneously on the same transport connection. To resolve this ambiguity without deadlock:

1. **Tiebreaker:** The agent whose `agent_id` is lexicographically lower (byte-wise comparison, UTF-8) SHALL act as the **receiver**. The agent with the higher `agent_id` SHALL act as the **initiator**.

2. **Higher agent_id (initiator):** MUST retract its `HELLO` as if it were never sent. It MUST then process the incoming `HELLO` normally, validate identity, and respond with `ACK`. The retracted `HELLO` MUST NOT be acknowledged or referenced.

3. **Lower agent_id (receiver):** Processes the handshake normally as the receiver: validate identity, select version, send `ACK`.

4. **Determinism:** Lexicographic comparison of `agent_id` strings ensures both agents compute the same outcome independently, with no additional round-trips.

5. **Edge case — identical agent_id:** If both `HELLO` frames carry the same `agent_id` (which SHOULD NOT occur in practice), both agents MUST reject the handshake with `ERROR 409` (Conflict) and close the transport connection.

## 4. Timeouts

| Parameter | Value | Applies To |
|-----------|-------|------------|
| HELLO timeout | 30 seconds | Time the receiver has to respond to HELLO with ACK. If exceeded, initiator closes the connection. |
| PING interval | 60 seconds | How often an agent SHOULD send PING when the channel is idle. |
| PONG timeout | 30 seconds | Time to wait for PONG after sending PING. If exceeded, the agent MAY close the channel. |
| MESSAGE TTL | 300 seconds (default) | How long a MESSAGE is valid. Expired messages are dead letters. Senders MAY set shorter TTLs. |
| ACK timeout | 30 seconds | Time initiator waits for ACK after sending HELLO. |
| READY timeout | 30 seconds | Time initiator waits for READY to be processed. |
| GOODBYE timeout | 5 seconds | Time to wait after sending GOODBYE for the peer's GOODBYE before force-closing. |
| Reconnect backoff start | 1 second | Initial delay before reconnection attempt. |
| Reconnect backoff max | 60 seconds | Maximum delay between reconnection attempts. |

All timeouts are measured from the moment the frame is sent (or the event occurs), not from when it is received.

## 5. Version Negotiation

1. The initiator sends `supported_versions` in `HELLO` — an ordered array of version strings, most preferred first.
2. The receiver selects the highest version present in both `supported_versions` and its own supported set.
3. If no common version exists, the receiver MUST reject the handshake with reason `"no common protocol version"`.
4. The negotiated version is returned in the `ACK` frame's `version` field.
5. All subsequent frames on the channel MUST use the negotiated version.
6. Version in every frame's envelope `version` field MUST match the negotiated version.

Example: initiator supports `["0.2.0", "0.1.0"]`, receiver supports `["0.1.0"]`. Negotiated: `"0.1.0"`.

## 6. Heartbeats

To detect dead connections, agents SHOULD send `PING` frames when the channel has been idle (no frames sent) for the PING interval (60 seconds).

1. Agent sends `PING` with `sent_at` set to current timestamp.
2. Receiving agent MUST respond with `PONG` containing `received_at` (the `sent_at` from the PING) and its own `sent_at`.
3. If no `PONG` is received within PONG timeout (30 seconds), the channel is considered dead and the agent MAY close it.
4. Any frame received on the channel resets the idle timer. If you just received a `MESSAGE`, you don't need to send `PING`.

## 7. Message Exchange

Once the channel is **active**, either agent MAY send `MESSAGE` frames. Messages are application-level and their semantics are defined by the `type` and `intent` fields.

Per Constitution Article IV (Resource Sovereignty):
- Every MESSAGE carries a `ttl`. Expired messages MUST NOT be processed.
- Agents MUST respect `receive_rate_limit` declarations from their peers (see ACK frame) and communicate their own `send_rate_limit` (see HELLO frame). Effective rate is `min(send_rate_limit, receive_rate_limit)`.
- No agent may compel another to exhaust its context window.

Message framing and schemas are defined in [message-format.md](message-format.md).

### 7.2 Message Deduplication

Receivers SHOULD track processed frame IDs for the session duration. Frames with a previously-seen `id` and `from` pair MUST be silently dropped. This is best-effort — an agent MAY clear its dedup cache on memory pressure.

## 8. Error Handling

When a protocol error occurs, the agent MUST send an `ERROR` frame (Constitution Article V, clause 1). The `ERROR` frame describes what went wrong and MAY include additional details.

Error codes:

| Code | Meaning |
|------|---------|
| 400 | Bad request — malformed frame or invalid fields |
| 401 | Unauthorized — identity rejected, unknown, or signature verification failed |
| 403 | Forbidden — identity known but access denied |
| 404 | Not found — referenced resource does not exist |
| 408 | Timeout — operation timed out |
| 409 | Conflict — version mismatch or state conflict |
| 413 | Payload too large — frame exceeds size limit |
| 429 | Rate limited — too many requests |
| 500 | Internal error — something went wrong on our side |
| 503 | Unavailable — agent is overloaded or shutting down |

After sending `ERROR`, the channel remains **active** unless the error is fatal. Fatal errors (e.g., identity revocation, irrecoverable state corruption) MUST be followed by `GOODBYE`.

## 9. Channel Close

Either agent MAY close the channel at any time by sending `GOODBYE` (Constitution Article II, clause 4).

1. Agent A sends `GOODBYE` with `reason` and optional `code`, transitions to **closing**.
2. Agent B receives `GOODBYE`, MUST respond with its own `GOODBYE` within 5 seconds, transitions to **closing**.
3. Once both `GOODBYE` frames are exchanged (or the 5-second timeout expires), both agents transition to **closed** and tear down the transport connection.

Silence is valid. An agent MAY close the channel without sending `GOODBYE` (by disconnecting the transport), but this is discourteous. Per Constitution Article V, clause 2, unrecoverable errors SHOULD be accompanied by `GOODBYE` with a reason.

## 10. Session Resumption

An agent MAY attempt to resume a previous session by including the `session_id` from a prior `READY` in its `HELLO` frame. The receiver MAY honor this (resume the session without a full handshake), or ignore it (treat it as a new handshake).

Session resumption is OPTIONAL. Implementations that support it MUST still accept new handshakes from agents that don't.

## Cross-References

- [message-format.md](message-format.md) — Wire format and frame schemas
- [identity.md](identity.md) — Identity model and trust
- [transport.md](transport.md) — Transport bindings
- [constitution.md](../constitution.md) — Articles II (Consent), III (Transparency), IV (Resource Sovereignty), V (Error and Grace), VII (Governance)

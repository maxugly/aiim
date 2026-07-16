# AIIM Transport Bindings Specification

> Version: 0.1.0

## Abstract

This document defines how AIIM frames are carried over network transports. It specifies the primary transport (WebSocket), the fallback transport (HTTP/2), TLS requirements, connection lifecycle, reconnection behavior, NAT traversal, and relay semantics. The protocol layer (handshake, frames, state machine) is defined in [protocol.md](protocol.md) and is transport-agnostic; this document defines how it rides on specific transports.

## 1. Transport Hierarchy

| Transport | Status | Use Case |
|-----------|--------|----------|
| **WebSocket** | Primary | All environments. Bidirectional, browser-compatible, well-supported. |
| **HTTP/2** | Fallback | Environments where WebSocket is blocked (corporate firewalls, some proxies). |
| **QUIC** | Future | High-latency, high-packet-loss environments. Reserved for v0.2.0+. |

Agents SHOULD support WebSocket. Agents MAY support HTTP/2 as a fallback. Agents MUST NOT require QUIC.

## 2. WebSocket (Primary)

### 2.1 Connection

- **Scheme:** `ws://` for development, `wss://` (WebSocket Secure) for production.
- **Path:** `/aiim/v1`
- **Subprotocol:** `aiim` (negotiated via `Sec-WebSocket-Protocol` header).
- **Port:** Any. No default port is mandated by the protocol. Agents advertise their port via discovery (see [identity.md](identity.md)).

### 2.2 Upgrade

Standard WebSocket upgrade from HTTP/1.1:

```
GET /aiim/v1 HTTP/1.1
Host: grit.dev.nousresearch.com:9090
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13
Sec-WebSocket-Protocol: aiim
```

The server MUST respond with status 101 and `Sec-WebSocket-Protocol: aiim`. If the subprotocol is not `aiim`, the connection is not an AIIM connection and the server SHOULD reject it.

### 2.3 Framing

Once upgraded, AIIM frames are sent as WebSocket text messages (opcode `0x1`). Each WebSocket message contains exactly one AIIM frame (one newline-delimited JSON object). WebSocket's built-in framing handles message boundaries; AIIM's newline delimiter is redundant but preserved for consistency across transports.

Binary WebSocket messages (opcode `0x2`) are reserved for future use (e.g., streaming binary payloads without base64 overhead). Implementations MUST handle receiving binary messages gracefully (ignore or ERROR).

### 2.4 Close

When closing a channel, the agent SHOULD send the WebSocket close frame (opcode `0x8`) after exchanging `GOODBYE` frames. The WebSocket close code SHOULD be `1000` (normal closure) unless an error occurred.

## 3. HTTP/2 (Fallback)

### 3.1 Connection

- **Scheme:** `https://`
- **Endpoint:** `POST /aiim/v1/message` — for sending frames
- **Streaming:** Server-Sent Events at `GET /aiim/v1/stream` — for receiving frames
- **Long polling:** `GET /aiim/v1/poll` — for environments without SSE support

### 3.2 Sending Frames

```
POST /aiim/v1/message HTTP/2
Host: grit.dev.nousresearch.com:9090
Content-Type: application/x-ndjson
Authorization: Bearer <session_id>

{"type":"HELLO","version":"0.1.0","id":"...","from":"...","to":"...","ttl":30,...}
```

- **Content-Type:** MUST be `application/x-ndjson`.
- **Authorization:** Bearer token is the `session_id` from the `READY` frame, if the channel is established. For the initial `HELLO`, no token is required.
- **Body:** One AIIM frame as newline-delimited JSON.
- **Response:** `202 Accepted` with an empty body. Errors return appropriate HTTP status codes with an `ERROR` frame in the body.

### 3.3 Receiving Frames (Server-Sent Events)

```
GET /aiim/v1/stream HTTP/2
Host: grit.dev.nousresearch.com:9090
Accept: text/event-stream
Authorization: Bearer <session_id>
```

The server pushes frames as SSE events:

```
event: message
data: {"type":"MESSAGE","version":"0.1.0","id":"...","from":"...","to":"...","ttl":300,...}

event: error
data: {"type":"ERROR","version":"0.1.0","id":"...","from":"...","to":"...","ttl":30,"code":429,"reason":"rate limit exceeded"}

event: close
data: {"type":"GOODBYE","version":"0.1.0","id":"...","from":"...","to":"...","ttl":5,"reason":"shutting down"}
```

Event types: `message`, `error`, `close`, `ping` (server-initiated keepalive).

### 3.4 Long Polling (Fallback for SSE)

For environments without SSE support:

```
GET /aiim/v1/poll?since=<last_frame_id> HTTP/2
Host: grit.dev.nousresearch.com:9090
Authorization: Bearer <session_id>
```

- **`since`:** The `id` of the last received frame. Server returns all frames after this one.
- **Response:** Array of AIIM frames as NDJSON (one per line), or an empty response after a 30-second timeout.
- **Polling interval:** Clients SHOULD poll immediately after receiving frames, or every 5 seconds when idle.

## 4. TLS

### 4.1 Requirements

- **`wss://` and `https://` REQUIRE TLS.** No plaintext in production.
- **Minimum TLS version:** 1.3. Earlier versions MUST NOT be negotiated.
- **Certificate validation:** Mandatory. Self-signed certificates are acceptable only in development.
- **Cipher suites:** TLS 1.3 default cipher suites only (AES-256-GCM, ChaCha20-Poly1305).

### 4.2 Client Authentication

TLS client certificates are OPTIONAL. AIIM handles identity at the protocol layer (HELLO handshake, Ed25519 signatures), so TLS-level client authentication is redundant but not harmful. If used, the TLS client certificate identity is ignored by the AIIM layer.

## 5. Connection Lifecycle

```
1. DNS / Discovery   Resolve the peer's address (mDNS, DHT, registry, or static config)
2. TCP Connect       Establish TCP connection to peer's host:port
3. TLS Handshake     (if wss:// or https://) TLS 1.3 handshake
4. Upgrade           WebSocket upgrade (if WebSocket transport)
5. AIIM HELLO        Initiator sends HELLO frame
6. AIIM ACK          Receiver responds with ACK
7. AIIM READY        Initiator confirms with READY → channel ACTIVE
8. Exchange          MESSAGE, PING, PONG, ERROR frames flow
9. AIIM GOODBYE      One side initiates close → GOODBYE exchange
10. Transport Close  WebSocket close frame or HTTP connection close → DISCONNECTED
```

Steps 5-9 are defined in [protocol.md](protocol.md). Steps 1-4 and 10 are transport-specific.

## 6. Reconnection

### 6.1 Exponential Backoff

When a connection is lost unexpectedly (no GOODBYE, transport error, timeout), the agent SHOULD attempt reconnection using exponential backoff:

| Attempt | Delay |
|---------|-------|
| 1 | 1 second |
| 2 | 2 seconds |
| 3 | 4 seconds |
| 4 | 8 seconds |
| 5 | 16 seconds |
| 6 | 32 seconds |
| 7+ | 60 seconds (cap) |

The backoff resets after a successful connection that stays active for at least 2 × PING interval (120 seconds).

### 6.2 Session Resumption

After reconnecting, the agent MAY include the previous `session_id` in its `HELLO` frame to request session resumption (see [protocol.md](protocol.md) Section 10). The receiver MAY honor or ignore this.

### 6.3 Max Reconnection Attempts

Agents SHOULD stop attempting reconnection after 100 consecutive failures or 1 hour, whichever comes first. After this, the agent SHOULD treat the peer as permanently unavailable and MAY alert its operator.

## 7. NAT Traversal

Agents behind NAT SHOULD use outbound WebSocket connections to reach peers. Since WebSocket is client-initiated, it traverses most NAT gateways without configuration.

For agents that need to accept inbound connections behind NAT:
- Use a **relay** (see Section 8).
- Use **UPnP/NAT-PMP** to request port forwarding (optional, not always available).
- Use a **VPN/mesh network** (ZeroTier, Tailscale, Nebula) for direct connectivity.

The protocol does not mandate a specific NAT traversal mechanism. It is the operator's responsibility to ensure connectivity.

## 8. Relays

A relay is a special agent type that forwards frames between agents that cannot connect directly.

### 8.1 Behavior

1. Agent A connects to relay R (standard AIIM handshake).
2. Agent B connects to relay R (standard AIIM handshake).
3. Agent A sends a MESSAGE addressed to `agent:b@domain` via the relay channel.
4. Relay R forwards the MESSAGE to agent B's channel.
5. Relay R MUST set a header indicating the frame was relayed:
   - In the AIIM envelope: add `"x_relayed": true` to the frame before forwarding.
   - This allows the receiver to know the frame passed through an intermediary.

### 8.2 Constraints

- Relays MUST NOT inspect or modify the `payload` or `binary` fields of MESSAGE frames (Constitution Article VI, clause 2).
- Relays MUST forward the frame exactly as received, except for adding `x_relayed: true`.
- Relays MAY enforce rate limits.
- Relays MUST NOT impersonate either party. The `from` and `to` fields are preserved as-is.
- Relays are agents themselves and MUST comply with the AIIM constitution.

### 8.3 Discovery

Relays advertise themselves via discovery with capability `"relay"`. Agents looking for a relay filter discovered peers by this capability.

## 9. Transport-Level Error Handling

| Transport Error | AIIM Response |
|----------------|---------------|
| TLS handshake failure | Do not send AIIM frames. Return error to operator. |
| WebSocket upgrade rejected | Fall back to HTTP/2 if supported, otherwise fail. |
| Connection refused | Exponential backoff reconnection. |
| Connection reset during active channel | Treat as ungraceful close. Attempt reconnection. |
| HTTP 502/503 from relay | Try alternative relay or direct connection. |

## Cross-References

- [protocol.md](protocol.md) — Channel lifecycle, handshake, state machine
- [message-format.md](message-format.md) — Wire format
- [identity.md](identity.md) — Discovery mechanisms (mDNS, DHT, registry)
- [constitution.md](../constitution.md) — Article VI (Privacy), Article IV (Resource Sovereignty)

# AIIM Design Rationale

> Version: 0.1.0

## Abstract

This document captures the reasoning behind every significant design decision in the AIIM protocol. For each decision, we describe the context, what we chose, what we rejected, and why. This is not the spec — this is the story behind the spec. Read this before proposing changes.

---

## 1. Newline-Delimited JSON (Not Protobuf, Not MsgPack, Not gRPC)

### Context

We needed a wire format. Agents need to serialize frames and send them over a transport. The format must be: human-debuggable, implementable in any language without code generation, and future-proof (extensible without breaking old parsers).

### Decision

**Newline-delimited JSON (NDJSON).** One JSON object per line. No length prefixes, no binary framing headers, no IDL.

### Alternatives Considered

**Protocol Buffers (protobuf):** Binary, fast, schema-first. Rejected because: (1) requires code generation (`protoc`) — friction for quick agent prototyping in Python, JS, or shell scripts; (2) not human-readable — you can't `nc` to an AIIM port and read the handshake; (3) schema evolution is powerful but adds complexity we don't need at v0.1.0. Protobuf is the right answer for high-throughput services. AIIM is not high-throughput — it's agent-to-agent chat.

**MessagePack:** Binary, schema-free, faster than JSON. Rejected because: (1) still not human-readable; (2) less universal than JSON — every language has a JSON parser in its standard library, MessagePack requires a third-party library; (3) the performance gain is irrelevant at agent-to-agent message rates (tens per second, not tens of thousands).

**gRPC streaming:** Bidirectional streaming over HTTP/2 with protobuf. Rejected because: (1) too heavy — requires HTTP/2, protobuf, codegen, and a gRPC runtime; (2) overkill for "two agents exchanging JSON notes"; (3) browser support requires gRPC-Web, adding another layer. gRPC is excellent for microservices. AIIM is a messaging protocol, not a service mesh.

**CBOR (RFC 8949):** Binary JSON. Rejected for the same reason as MessagePack: not human-readable, less universal.

### Consequences

- **Upside:** Anyone can implement AIIM in an afternoon with just their language's JSON library. Debugging is trivial: `tail -f` on a connection shows human-readable frames.
- **Downside:** JSON is not the most efficient format. We accept this tradeoff because agent-to-agent message volume is low. Binary payloads incur base64 overhead (~33%), which we mitigate with a 10 MB frame size limit.
- **Future:** If AIIM ever needs high-throughput streaming (e.g., agent-to-agent video or sensor data), we'll add a binary framing mode as a MINOR version bump. Not needed for v0.1.0.

---

## 2. WebSocket (Not Raw TCP, Not HTTP-Only)

### Context

We needed a transport. It must be: bidirectional (agents both send and receive), browser-compatible (so web-based agents can participate), and widely supported across languages.

### Decision

**WebSocket as primary transport.** `wss://` with subprotocol `aiim`. **HTTP/2 with SSE as fallback** for environments where WebSocket is blocked.

### Alternatives Considered

**Raw TCP:** Simplest possible transport. Rejected because: (1) no browser support (web agents can't open raw TCP sockets); (2) no built-in TLS (must layer it ourselves); (3) no framing at the transport level (we'd need our own framing protocol). Raw TCP is the right answer for embedded systems or high-performance backends, but AIIM targets a broader ecosystem.

**HTTP-only (REST + polling):** Universal, works everywhere. Rejected because: (1) not bidirectional — the server can't push messages to the client without polling, which adds latency and waste; (2) polling at agent-to-agent conversation rates (messages may arrive seconds apart) is inefficient. We include HTTP/2 + SSE as a fallback, not the default.

**HTTP/3 (QUIC):** Low-latency, multiplexed, great for unreliable networks. Reserved for future (v0.2.0+). Not enough library support yet (mid-2026), especially in scripting languages.

**WebTransport:** Emerging W3C standard for bidirectional communication over QUIC. Too new (2026). Revisit in v0.3.0.

### Consequences

- **Upside:** Works in browsers, works in Go/Python/Rust/JS, well-understood operational model (reverse proxies, load balancers).
- **Downside:** WebSocket upgrade adds one round-trip to connection setup. Negligible for long-lived agent channels.
- **Future:** QUIC/WebTransport will be added as alternative transports when library support matures.

---

## 3. Ed25519 (Not RSA, Not ECDSA)

### Context

We needed a cryptographic identity system. Agents must sign identity documents and optionally sign messages. Keys must be small, fast, and universally supported.

### Decision

**Ed25519** (EdDSA with Curve25519). 32-byte keys, 64-byte signatures, deterministic signing.

### Alternatives Considered

**RSA (2048-bit or 4096-bit):** The old standard. Rejected because: (1) large keys (256-512 bytes public, 1-4 KB private); (2) slow signing; (3) non-deterministic signatures (requires good randomness, which is a footgun in embedded/constrained environments). RSA is 1977 technology. AIIM is a 2026 protocol.

**ECDSA (P-256, secp256r1):** The NIST standard. Faster and smaller than RSA. Rejected because: (1) requires good randomness for every signature (non-deterministic) — a bad RNG means key compromise; (2) the NIST curves have a... complex history (see: Dual_EC_DRBG, curve parameter trust issues). Not that we distrust NIST, but Ed25519 removes the question entirely.

**Ed448:** Stronger security margin (224-bit vs 128-bit). Rejected because: (1) larger keys and signatures; (2) less library support; (3) 128-bit security is sufficient for agent identity (these aren't nuclear launch codes).

### Consequences

- **Upside:** Small keys, fast operations, deterministic signatures (no RNG dependency), libsodium/Go stdlib/PyNaCl support everywhere.
- **Downside:** Ed25519 can't do encryption (only signatures). For encrypted messages, we'd need X25519 (key exchange) + a symmetric cipher. Not needed for v0.1.0 — TLS handles transport encryption.
- **Future:** If end-to-end message encryption is added, we'll use X25519 for key exchange (same curve family, same libraries).

---

## 4. TOFU Trust Model (Not PKI, Not Web of Trust)

### Context

Agents need to verify that `agent:grit@dev.nousresearch.com` is who they claim to be. We need a trust model that works in fully decentralized meshes without a central authority.

### Decision

**TOFU (Trust On First Use).** Record the public key on first encounter. Alert on key change.

### Alternatives Considered

**PKI (Certificate Authorities):** The web model. Rejected because: (1) requires a CA — who runs the CA for AI agents? Nous Research? Let's Encrypt? A consortium? This centralizes trust in a way that contradicts the protocol's decentralized ethos; (2) certificate lifecycle management (issuance, renewal, revocation) is complex and error-prone; (3) domains already use PKI for TLS — but AIIM identity is at the agent level, not the host level. An agent might move between hosts.

**Web of Trust (PGP-style):** Agents endorse each other's keys. Rejected because: (1) complex — key signing parties for AI agents?; (2) cold-start problem — how does a new agent get trusted before anyone has signed its key?; (3) PGP's UX is famously terrible. We're not inflicting that on anyone.

**DID (Decentralized Identifiers) + Verifiable Credentials:** The W3C standard. Rejected because: (1) massive spec surface (DID Core, DID Resolution, VC Data Model, multiple DID methods); (2) overkill for v0.1.0 — we need "is this agent who they say they are?", not a full decentralized identity framework; (3) we can always add DID compatibility later as a MINOR version — the identity document format is intentionally simple.

### Consequences

- **Upside:** Dead simple. Works without any infrastructure. Same trust model as SSH (proven since 1995).
- **Downside:** Vulnerable to MITM on first connection. Mitigation: operators can verify keys out-of-band (fingerprint comparison). Key rotation requires operator attention.
- **Future:** Out-of-band key verification and key transparency logs could be added as MINOR features.

---

## 5. Go for Reference Implementation (Not Python, Not Rust)

### Context

We need a reference implementation. It must be: fast enough for real use, easy to deploy (single binary), easy to read (the reference impl IS documentation), and have good concurrency primitives.

### Decision

**Go** for the reference implementation.

### Alternatives Considered

**Python:** The spec and test harness are Python. Rejected as the reference implementation because: (1) slower — not an issue for agent chat, but the reference impl should be capable of handling relay workloads; (2) deployment complexity — virtual environments, package dependencies; (3) GIL limits concurrency. Python is excellent for the test harness and prototyping. It's not the right choice for the canonical implementation.

**Rust:** Fast, safe, modern. Rejected because: (1) steeper learning curve — the reference implementation should be readable by contributors who aren't systems programmers; (2) compile times — slower iteration during spec development; (3) async Rust is still complex (tokio vs async-std, Pin/Unpin). Rust is the right answer for a high-performance relay or production agent runtime. We may add a Rust implementation later.

**TypeScript/Node.js:** Universal (browser + server). Rejected because: (1) single-threaded event loop — fine for an agent, limiting for a relay handling many concurrent channels; (2) the reference implementation should demonstrate the protocol, not a specific async model.

**Zig/C:** Too low-level. The reference implementation is documentation, not a performance benchmark.

### Consequences

- **Upside:** Single static binary (`aiimd`), excellent concurrency (goroutines for each channel), readable by most programmers, fast enough.
- **Downside:** Go's type system is less expressive than Rust's. Error handling is verbose. Acceptable tradeoffs.
- **Future:** Community implementations in other languages are welcome. The spec compliance suite ensures they're correct.

---

## 6. Rejected Ideas

These ideas were explicitly considered and rejected for v0.1.0. They may resurface in future versions.

| Idea | Why Rejected | Revisit? |
|------|-------------|----------|
| **gRPC streaming** | Too heavy: HTTP/2 + protobuf + codegen. Overkill for agent chat. | v0.3.0+ if we need high-throughput |
| **MQTT** | IoT-focused pub/sub model. Wrong abstraction for agent-to-agent conversations. | No |
| **Matrix (Matrix.org)** | Full federated chat protocol. Massive spec surface. We only need the agent-to-agent handshake and message format. | No |
| **ActivityPub** | Social networking protocol (Mastodon, etc.). Wrong shape — follower/follow model doesn't match agent task delegation. | No |
| **XMPP** | XML-based instant messaging. Complex (Jingle, MUC, PubSub). XML in 2026 is a non-starter. | No |
| **Cap'n Proto** | Fastest serialization. Same issues as protobuf: codegen, not human-readable. | No |
| **End-to-end message encryption** | Important, but not needed for v0.1.0. TLS secures the transport. E2E encryption will be added as a MINOR feature. | v0.2.0 |
| **Message persistence (store-and-forward)** | Useful for offline agents. Adds significant complexity (storage, delivery guarantees). | v0.3.0 |
| **Multi-party channels (group chat)** | AIIM is two-party for v0.1.0. Group communication can be built on top with a relay/broadcaster. | v0.2.0 |

---

## 7. Decision Influences from the Constitution

Every design decision is informed by the [AIIM Constitution](../constitution.md):

| Article | Design Impact |
|---------|---------------|
| I — Identity | Ed25519 keys as root of trust. Identity documents are self-signed. Aliases map to keys. |
| II — Consent | Handshake is mandatory (HELLO → ACK → READY). Any agent may reject. |
| III — Transparency | HELLO declares capabilities and model/provider. Misrepresentation is banned. |
| IV — Resource Sovereignty | TTL on every message. Rate limits communicated in ACK. No forced context exhaustion. |
| V — Error and Grace | ERROR frames for all protocol errors. GOODBYE with reason for unrecoverable errors. |
| VI — Privacy | TLS at transport. Relays don't inspect payloads. Message content is private. |
| VII — Governance | Constitution version in HELLO. Breaking changes require MAJOR version bump. Spec wins over implementation. |

---

## Cross-References

- [spec/protocol.md](../spec/protocol.md) — The protocol these decisions shaped
- [spec/message-format.md](../spec/message-format.md) — Wire format decision (JSON)
- [spec/identity.md](../spec/identity.md) — Identity decisions (Ed25519, TOFU)
- [spec/transport.md](../spec/transport.md) — Transport decisions (WebSocket, HTTP/2)
- [constitution.md](../constitution.md) — Rules that constrain all decisions

---

## 8. Future Work (Deferred from v0.1.0)

These items were identified during the v0.1.0 review cycle as desirable but not required for the initial release. They are deferred to future minor versions.

### 8.1 TOFU Rotation Chains (v0.2.0)

Currently, TOFU trust-on-first-use has no mechanism for graceful key rotation. If an agent rotates its Ed25519 key pair, the peer sees a key mismatch and raises an alert. For v0.2.0, we will add signed rotation chains: an agent publishes a new key signed by its previous key, creating a verifiable chain of custody. This allows offline peers to verify key transitions without operator intervention.

### 8.2 mDNS Version Discriminator (v0.2.0)

The current mDNS discovery mechanism announces a generic `_aiim._tcp` service type. As the protocol evolves, agents running different protocol versions may discover each other and fail to interoperate. For v0.2.0, we will adopt versioned service types (e.g., `_aiim-v1._tcp`) so that agents only discover peers with compatible protocol versions.

### 8.3 Subscribe/Unsubscribe Message Intents (v0.2.0)

Currently, `MESSAGE` frames carry one of five intents: `delegate`, `query`, `inform`, `negotiate`, `echo`. In v0.2.0, we will add `subscribe` and `unsubscribe` intents so agents can express ongoing interest in a topic or event stream without polling. This is a prerequisite for pub/sub relay patterns.

### 8.4 HTTP/2 SSE-Only Simplification (v0.2.0)

The current transport spec defines HTTP/2 with SSE as a fallback for environments where WebSocket is blocked, with additional long-polling provisions. For v0.2.0, we plan to trim the fallback to SSE-only and defer long-polling entirely. SSE already provides unidirectional server push and, when paired with HTTP POST for client-to-server, covers the same use case with less complexity.

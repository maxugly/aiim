# AGENTS.md — impl/reference/ Directory

> *For autonomous agents working on the Go reference implementation. Humans: this is the canonical AIIM implementation.*

## Overview

This directory contains the Go reference implementation of the AIIM protocol. It is the **canonical implementation** — the first implementation, the one that validates the spec, and the one all other implementations are tested against. It is not necessarily the fastest or the prettiest, but it is correct.

## Language

- **Language:** Go 1.22+
- **Module:** `github.com/nousresearch/aiim` (pending)
- **License:** MIT (pending)

## Package Structure

```
impl/reference/
├── AGENTS.md              — This file
├── VERSION                — "0.1.0"
├── go.mod                 — Go module definition
├── go.sum                 — Dependency checksums
├── cmd/
│   └── aiimd/             — AIIM daemon: runs an agent, listens for connections
│       └── main.go        — Entrypoint
├── pkg/
│   ├── protocol/          — Core protocol logic
│   │   ├── frame.go       — Frame types, JSON marshal/unmarshal
│   │   ├── state.go       — Channel state machine
│   │   ├── handshake.go   — HELLO/ACK/READY handshake logic
│   │   └── frame_test.go  — Unit tests
│   ├── identity/          — Identity management
│   │   ├── keypair.go     — Ed25519 key generation, signing, verification
│   │   ├── document.go    — Identity document creation, validation
│   │   └── identity_test.go
│   ├── transport/         — Transport bindings
│   │   ├── websocket.go   — WebSocket client/server
│   │   ├── http.go        — HTTP/2 fallback transport
│   │   └── transport_test.go
│   └── discovery/         — Agent discovery
│       ├── mdns.go        — mDNS (_aiim._tcp) service advertising/discovery
│       ├── dht.go         — Kademlia DHT for mesh discovery
│       ├── registry.go    — Optional HTTPS registry client
│       └── discovery_test.go
├── internal/              — Private packages (no external imports)
│   ├── wire/              — Low-level wire format helpers
│   └── util/              — Internal utilities
└── compliance/            — Spec compliance test suite
    ├── suite.go           — Test runner
    ├── handshake_test.go  — Handshake compliance tests
    ├── framing_test.go    — Wire format compliance tests
    └── transport_test.go  — Transport compliance tests
```

## Build Instructions

```bash
# Build the AIIM daemon
cd impl/reference
go build ./cmd/aiimd/

# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run spec compliance suite (requires two running agents)
go test -tags=compliance ./compliance/ -v
```

## Test Commands

```bash
# Unit tests only (fast)
go test -short ./...

# Integration tests (requires network)
go test ./pkg/transport/ -v

# Compliance suite (requires two agent instances)
go test -tags=compliance ./compliance/ -v -count=1

# Test coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Conventions

### Code conventions

1. **Standard Go conventions.** Follow `gofmt`, `golint`, and `go vet`. No custom formatting.
2. **One package per concept.** Protocol state machine in `pkg/protocol/`, identity in `pkg/identity/`, etc.
3. **Public API is minimal.** Only export what other packages genuinely need. Everything else goes in `internal/`.
4. **Interfaces for testability.** Transport, discovery, and identity operations are defined as interfaces so they can be mocked.

### Testing conventions

5. **Tests live alongside code.** `frame.go` → `frame_test.go`. No separate `test/` directory.
6. **Table-driven tests.** Go style: define test cases as a slice of structs, iterate.
7. **Error messages include context.** `t.Errorf("handshake failed: %v", err)` not `t.Error("handshake failed")`.
8. **Compliance tests are language-agnostic.** They connect to a running agent over the wire, not by importing Go packages. Any implementation in any language can be tested.

### Protocol compliance

9. **The reference implementation MUST exactly match the spec.** No deviations, no "optimizations" that change behavior.
10. **If the spec is ambiguous, the reference implementation clarifies it.** Document the clarification in the spec.
11. **Frame parsing is strict.** Unknown fields are rejected, not ignored (fail closed).

### Error handling

12. **Every error is wrapped with context.** Use `fmt.Errorf("reading HELLO frame: %w", err)`.
13. **Protocol errors generate ERROR frames.** Internal errors generate GOODBYE frames with a reason.
14. **Never panic.** Return errors up the stack. The daemon handles graceful shutdown.

### Performance

15. **Correctness before performance.** Don't optimize until the compliance suite passes.
16. **Zero-allocation where practical.** Frame parsing should avoid unnecessary allocations.
17. **Concurrency-safe.** Channel state machine must be safe for concurrent access (goroutines).

## Dependencies

| Package | Purpose | Why |
|---------|---------|-----|
| `golang.org/x/crypto` | Ed25519 | Standard Ed25519 implementation |
| `github.com/gorilla/websocket` | WebSocket | Well-maintained, spec-compliant |
| `github.com/google/uuid` | UUIDv4 generation | Standard, no dependencies |
| `github.com/hashicorp/mdns` | mDNS discovery | Battle-tested |

No other dependencies without a documented rationale in `design/`.

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2026-07-16 | Initial directory structure and conventions |

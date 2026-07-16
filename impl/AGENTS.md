# AGENTS.md — impl/ Directory

> *For autonomous agents writing AIIM implementations. Humans: this is where the spec becomes code.*

## What Lives Here

Every subdirectory of `impl/` contains a reference or community implementation of the AIIM protocol in a specific language. The `reference/` directory holds the Go reference implementation — the canonical code that validates the spec.

## Conventions

### Language directories

1. **Each language gets its own directory:** `impl/reference/` (Go), `impl/python/` (Python test harness), `impl/rust/` (future), etc.
2. **Every language directory has its own AGENTS.md** with language-specific conventions, build instructions, and test commands.
3. **New language implementations are welcome** but must pass the spec compliance suite before being merged.

### Testing

4. **Every implementation MUST pass the spec compliance suite.** The suite lives in `impl/reference/compliance/` and is language-agnostic (it exercises the protocol over the wire).
5. **Unit tests are REQUIRED.** Every exported function, every frame parser, every state transition.
6. **Integration tests are REQUIRED.** At minimum: happy-path handshake, timeout handling, error frame propagation, reconnection.

### Compliance with spec

7. **Implementations follow the spec, not each other.** If `impl/reference/` has a bug, fix the bug — don't copy it into other implementations.
8. **Frame types, field names, and wire format MUST exactly match the spec.** No aliases, no shortcuts, no "but it's faster this way."
9. **Constitution compliance is mandatory.** Every implementation MUST enforce consent (handshake required), transparency (capability declaration), and resource sovereignty (TTL, rate limits).

### Code quality

10. **Follow the language's standard conventions.** Go: `gofmt`, `golint`, standard project layout. Python: `black`, `mypy`, `pytest`.
11. **Document public APIs.** Every exported symbol gets a doc comment.
12. **Errors are meaningful.** No `error: something went wrong`. Include context: `handshake timeout: expected ACK from agent:grit@dev.nousresearch.com within 30s`.

### Versioning

13. **Implementation versions track spec versions.** `impl-v0.1.0` implements `spec-v0.1.0`.
14. **Version is declared in a VERSION file** at the root of each language directory.

### CI/CD

15. **All implementations MUST have a CI pipeline.**
16. **CI runs:** lint, unit tests, integration tests, spec compliance suite.
17. **Breaking spec changes mean breaking CI.** That's expected — update the implementation to match.

## Directory Structure

```
impl/
├── AGENTS.md              — This file
├── reference/             — Go reference implementation
│   ├── AGENTS.md          — Reference impl conventions
│   ├── VERSION            — "0.1.0"
│   ├── go.mod
│   ├── cmd/               — CLI entrypoints
│   ├── pkg/               — Library packages
│   │   ├── protocol/      — Frame parsing, state machine
│   │   ├── identity/      — Key generation, identity docs
│   │   ├── transport/     — WebSocket, HTTP/2 bindings
│   │   └── discovery/     — mDNS, DHT, registry client
│   ├── internal/          — Private implementation details
│   └── compliance/        — Spec compliance test suite
└── python/                — Python test harness (future)
    ├── AGENTS.md
    └── VERSION
```

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2026-07-16 | Initial directory conventions |

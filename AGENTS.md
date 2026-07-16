# AGENTS.md — AIIM Repository

> *For autonomous agents working in this repo. Humans: this is how the robots understand the project. Read it anyway.*

## Project Identity

- **Name:** AIIM (Artificial Intelligence Instant Messenger)
- **Type:** Protocol specification + reference implementation
- **Language:** English (specs), Go (reference impl), Python (test harness)
- **Version:** 0.1.0-prealpha

## Repository Layout

```
aiim/
├── README.md           — Project overview, status, directory map
├── constitution.md     — Rules of engagement for AIIM agents
├── AGENTS.md           — This file
├── spec/               — Protocol specification documents
│   ├── AGENTS.md       — Spec directory conventions
│   ├── protocol.md     — Core protocol: handshake, lifecycle, framing
│   ├── message-format.md — Wire format: JSON schema, types, versioning
│   ├── identity.md     — Identity model: keys, discovery, trust
│   └── transport.md    — Transport bindings: WebSocket, HTTP, QUIC
├── impl/               — Reference implementations
│   ├── AGENTS.md       — Implementation conventions
│   └── reference/      — Go reference implementation
│       └── AGENTS.md   — Reference impl specifics
├── design/             — Design decisions and rationale
│   ├── AGENTS.md       — Design doc conventions
│   └── rationale.md    — Why we chose what we chose
└── docs/               — User-facing documentation
    └── AGENTS.md       — Docs conventions
```

## Conventions

### For all agents working in this repo:

1. **Read the spec first.** The spec directory is authoritative. Implementation follows spec, not the other way around.
2. **AGENTS.md in every directory.** Every directory has one. Read it before touching files there.
3. **Spec changes require a proposal.** Don't edit `spec/` files directly without documenting the rationale in `design/`.
4. **Constitution compliance is mandatory.** Any agent behavior described here MUST comply with the AIIM constitution.
5. **YAGNI.** If we don't need it for v0.1.0, it goes in `design/rejected.md`, not in the spec.
6. **Reference impl is Go.** The `impl/reference/` directory is Go. Other languages go in `impl/<lang>/`.

### Naming conventions:

- Files: `lowercase-hyphenated.md` for docs, `lowercase_snake.go` for Go
- Protocol frames: `UPPER_SNAKE_CASE` (e.g., `HELLO`, `GOODBYE`, `MESSAGE`)
- JSON fields: `snake_case`
- Version: SemVer (`MAJOR.MINOR.PATCH`)

### Commit conventions:

- `spec:` — specification changes
- `impl:` — implementation changes
- `design:` — design doc changes
- `docs:` — documentation changes
- `chore:` — repo maintenance

## Current State

**Phase:** Pre-alpha specification

**What exists:**
- This AGENTS.md
- README.md (overview)
- constitution.md (agent rules of engagement)

**What needs to exist:**
- `spec/protocol.md` — Core protocol spec
- `spec/message-format.md` — Wire format spec
- `spec/identity.md` — Identity model spec
- `spec/transport.md` — Transport binding spec
- `design/rationale.md` — Design decisions
- `impl/reference/` — Go reference implementation
- Every directory needs its own AGENTS.md

**Next action:** Write the spec sheets. Then implement.

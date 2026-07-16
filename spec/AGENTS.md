# AGENTS.md — spec/ Directory

> *For autonomous agents writing protocol specs. Humans: this is where the truth lives.*

## What Lives Here

Every `.md` file in `spec/` defines a normative part of the AIIM protocol. These documents are **authoritative** — implementation follows spec, not the other way around. If the reference implementation and the spec disagree, the spec wins (Constitution Article VII, clause 4).

## Conventions

### Authoritativeness

1. **Spec is the source of truth.** Implementation bugs are bugs. Spec bugs are spec bugs — fix the spec, then fix the implementation.
2. **No ambiguity.** Every requirement MUST use RFC 2119 keywords: MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, MAY, OPTIONAL.
3. **Testable.** Every normative statement SHOULD be verifiable by an automated compliance suite.

### Versioning

4. **Every spec file carries a version banner** at the top, after the title: `> Version: 0.1.0`
5. **Version is SemVer** (MAJOR.MINOR.PATCH):
   - MAJOR: breaking wire-format or protocol changes
   - MINOR: additive features, new frame types, new fields
   - PATCH: clarifications, typo fixes, non-normative improvements
6. **All spec files in a directory share the same version.** Bumping one bumps all. A spec release is atomic.

### Naming

7. **Frame types are UPPER_SNAKE_CASE** (e.g., `HELLO`, `ACK`, `READY`, `MESSAGE`, `ERROR`, `GOODBYE`, `PING`, `PONG`).
8. **JSON fields are snake_case** (e.g., `agent_id`, `supported_versions`, `constitution_version`).
9. **Files are lowercase-hyphenated.md** (e.g., `protocol.md`, `message-format.md`, `identity.md`, `transport.md`).

### Content structure

10. **Every spec file follows this structure:**
    - Title + version banner
    - Abstract (one paragraph: what this spec defines and why)
    - Normative sections with RFC 2119 language
    - Examples (annotated JSON or diagrams)
    - Cross-references to other spec files and constitution articles

### Change process

11. **Spec changes require a design proposal.** Before editing a spec file, create or update the corresponding rationale in `design/`.
12. **Backward compatibility is the default.** Prefer additive changes over breaking ones.
13. **Deprecation before removal.** Remove a field/frame type only after one major version of deprecation.

### Cross-references

14. **Link to other spec files** using relative paths: `[protocol.md](protocol.md)`.
15. **Reference constitution articles** by article number: "Constitution Article II (Consent)".
16. **Never duplicate normative text.** If another spec file defines something, link to it — don't copy-paste.

### Diagrams

17. **State machines use ASCII art.** No external diagram tools, no image files. Every agent must be able to render and understand them.
18. **Example frames use annotated JSON** with comments explaining each field inline.

## Files

| File | What it defines |
|------|----------------|
| `protocol.md` | Core protocol: handshake, channel lifecycle, state machine, timeouts |
| `message-format.md` | Wire format: envelope structure, frame schemas, JSON Schema, binary payloads |
| `identity.md` | Identity model: key material, identity documents, discovery, trust model |
| `transport.md` | Transport bindings: WebSocket, HTTP/2, QUIC, TLS, reconnection, relays |

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2026-07-16 | Initial pre-alpha specification |

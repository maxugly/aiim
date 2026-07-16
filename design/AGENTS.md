# AGENTS.md — design/ Directory

> *For autonomous agents documenting design decisions. Humans: this is where we explain why, not just what.*

## What Lives Here

Every `.md` file in `design/` captures the reasoning behind a protocol decision. These documents answer "why did we choose X over Y?" so future maintainers (and future agents) don't retrace our steps.

## Conventions

### Every decision MUST reference alternatives

1. **For every design choice documented, list at least two alternatives considered.** If you didn't consider alternatives, you didn't design — you guessed.
2. **For each alternative, state why it was rejected.** "It's worse" is not a reason. Be specific: performance, complexity, ecosystem compatibility, YAGNI, etc.
3. **Link alternatives to concrete technologies** where relevant (e.g., "we considered protobuf" not "we considered binary formats").

### YAGNI enforcement

4. **If it's not needed for v0.1.0, it doesn't go in the spec.** Ideas for v0.2.0+ go here in `design/rejected.md` or `design/future.md`.
5. **"We might need this later" is not a reason to add something now.** The spec is minimal by design.
6. **Every field in every frame MUST be justified in a design doc.** If you can't explain why a field exists, it shouldn't.

### Document structure

7. **Every design doc follows this structure:**
   - Title + version banner
   - Context (what problem are we solving?)
   - Decision (what did we choose?)
   - Alternatives considered (what did we reject and why?)
   - Consequences (what tradeoffs does this decision create?)
   - Cross-references to spec files and constitution articles

### Tone

8. **Design docs are conversational but precise.** Write like you're explaining your choices to a colleague at 2 a.m.
9. **No marketing, no hype.** Just reasoning.
10. **Acknowledge downsides.** Every decision has tradeoffs. Pretending otherwise erodes trust.

### Versioning

11. **Design docs are versioned alongside the spec they inform.** If you change a design doc to support a new spec version, bump the version banner.
12. **If a previous decision is reversed, document the reversal** rather than overwriting history. Add a "Reversal" section with date and reason.

### Cross-references

13. **Link to the spec files your decision affects.**
14. **Reference rejected ideas by name.** "See `design/rejected.md#grpc-streaming`."
15. **Reference constitution articles** where relevant.

## Files

| File | What it covers |
|------|---------------|
| `rationale.md` | Core design decisions: JSON vs binary, WebSocket vs TCP, Ed25519 vs RSA, TOFU vs PKI, Go vs Python/Rust |
| `rejected.md` | Ideas considered and rejected (future) |
| `future.md` | Ideas deferred to future versions (future) |

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2026-07-16 | Initial directory conventions |

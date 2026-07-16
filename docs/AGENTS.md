# AGENTS.md — docs/ Directory

> *For autonomous agents writing user-facing documentation. Humans: this is where you learn how to use AIIM.*

## What Lives Here

Every `.md` file in `docs/` is user-facing documentation: guides, tutorials, examples, and reference material for developers integrating AIIM into their agents. These files are **not normative** — the spec is in `spec/`. Docs explain how to use what the spec defines.

## Conventions

### Audience

1. **Write for a developer who has never read the spec.** Assume they know Go (or their language) but not AIIM.
2. **Start every guide with prerequisites.** What do they need installed? What do they need to know?
3. **Use "you" and "your agent"** — docs are addressed to the implementor.

### Structure

4. **Guides are step-by-step.** Numbered steps with code snippets that can be copy-pasted.
5. **Examples are runnable.** Every code example MUST be tested against the reference implementation before committing. If it doesn't compile, it doesn't ship.
6. **Tutorials have an end state.** The reader should have something working by the end.

### Completeness

7. **Every doc has:**
   - Title + brief description
   - Prerequisites
   - Step-by-step instructions
   - Complete, runnable code examples
   - Expected output
   - Troubleshooting section (common errors and fixes)
   - Links to related docs and spec files

### Maintenance

8. **Docs MUST stay in sync with the reference implementation.** If the API changes, update the docs in the same commit.
9. **If a code example breaks, fix it immediately.** Broken examples are worse than no examples.
10. **Date every doc.** Include a "Last updated" footer so readers know how fresh it is.

### Tone

11. **Friendly but professional.** AIIM is fun, but docs should be clear first, clever second.
12. **Explain jargon on first use.** Not everyone knows what "newline-delimited JSON" means.

### What NOT to put here

13. **Don't duplicate the spec.** Link to `spec/` for normative details.
14. **Don't duplicate the design rationale.** Link to `design/` for why decisions were made.
15. **Don't put implementation internals here.** That goes in `impl/` AGENTS.md or package godoc.

## Files (planned)

| File | Audience | Content |
|------|----------|---------|
| `getting-started.md` | New AIIM developers | Install, first handshake, send a message |
| `agent-guide.md` | Agent authors | Identity setup, channel management, message types |
| `transport-setup.md` | Infrastructure devs | WebSocket config, TLS, relays, NAT traversal |
| `examples/` | Everyone | Runnable example agents in Go and Python |

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2026-07-16 | Initial directory conventions |

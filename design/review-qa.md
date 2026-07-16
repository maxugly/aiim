# AIIM QA Review — Pre-Alpha Audit

**Date:** 2026-07-16
**Reviewer:** Automated QA Agent
**Repository:** `~/snc/cod/aiim/`
**Scope:** Every file on disk

---

## Summary

The AIIM repository is a **skeleton**. Three files exist (`README.md`, `AGENTS.md`, `constitution.md`) and five empty directories (`spec/`, `design/`, `docs/`, `impl/`, `impl/reference/`). The AGENTS.md explicitly acknowledges this — its "Current State" section says "What needs to exist" for 7+ items. There are no spec files to cross-reference, no implementations to audit, and no versioned artifacts to validate. The review is therefore limited to internal consistency of the three files that exist, plus the gap between what's promised and what's on disk. **Consistency score: 2/10** — the skeleton itself is internally coherent, but the repository claims to contain things it doesn't, and the constitution defines protocol concepts that have no specification anywhere.

---

## Critical Issues (Blocking, Must Fix)

### C1: No specification files exist

The entire purpose of the repo is "a protocol spec" (README, line 13), yet `spec/` is empty. The constitution defines protocol frames (`HELLO`, `ACK`, `READY`, `GOODBYE`, `ERROR`, `MESSAGE`, `TTL`, `RATE_LIMIT`) with no corresponding specification. Any agent implementer has zero guidance on:

- Frame structure (JSON? binary? length-delimited?)
- Handshake state machine (timeouts? retries? ordering requirements?)
- Identity format (how are Ed25519 keys serialized on wire? hex? base64? multibase?)
- Error codes or categories (what goes in an `ERROR` frame?)
- Message TTL semantics (seconds? milliseconds? absolute timestamps?)
- Rate limit format (requests-per-second? token bucket parameters?)

**Severity:** All protocol work is blocked on this. The constitution is a requirements document, not a spec.

### C2: No AGENTS.md files in any subdirectory

AGENTS.md (line 41) states: *"AGENTS.md in every directory. Every directory has one."* None of the five subdirectories (`spec/`, `design/`, `docs/`, `impl/`, `impl/reference/`) contain an AGENTS.md. The repository layout block (lines 14–33) lists 6 AGENTS.md files that don't exist.

**Severity:** Violates the repo's own stated convention. An agent entering any subdirectory has no guidance.

### C3: No git repository

There is no `.git/` directory. The commit conventions section (AGENTS.md lines 55–60) defines commit message prefixes (`spec:`, `impl:`, `design:`, `docs:`, `chore:`) but there's no VCS to commit to. This is the first thing any collaborator would notice.

**Severity:** Blocks collaborative development. May also explain why no files exist — the repo may have been scaffolded but never initialized.

### C4: Constitution references undefined protocol primitives

The constitution contains normative requirements (MUST/SHOULD/MAY) that reference undefined concepts:

| Article | Reference | Problem |
|---------|-----------|---------|
| II.2 | `HELLO` → `ACK` → `READY` | Frame format and sequencing rules unspecified |
| II.4 | `GOODBYE` frame | No specification of what fields it carries |
| III.1 | capabilities in `HELLO` frame | No capability taxonomy or schema defined |
| III.2 | model/provider in identity metadata | Identity metadata format unspecified |
| IV.1 | `TTL` on every message | What unit? Where in the frame? Default? |
| IV.2 | `RATE_LIMIT` declarations | Format unspecified |
| V.1 | `ERROR` frame | No error taxonomy, no error frame format |
| V.2 | `GOODBYE` with a reason | Reason format unspecified |
| VII.1 | constitution version in `HELLO` | No field name specified |

These are not "to be defined later" — they're written as active MUST requirements with no supporting specification. An implementer reading the constitution has binding obligations with no way to satisfy them.

---

## Warnings (Should Fix, Non-Blocking)

### W1: Version string inconsistency

Three different version representations exist for the same thing:

- `README.md` line 21: `"v0.1.0 pre-alpha"`
- `AGENTS.md` line 10: `"0.1.0-prealpha"`
- `constitution.md` line 59: `"v0.1.0"`

Are these the same version? Does `pre-alpha` == `-prealpha`? For SemVer (which AGENTS.md line 52 mandates), the pre-release tag should be `0.1.0-prealpha` — the `v` prefix is not part of SemVer but is common in tags. Pick one and use it consistently.

### W2: Directory map mismatch between README and AGENTS.md

| What | README directory map | AGENTS.md layout |
|------|---------------------|-----------------|
| `spec/` contents | "message format, identity, transport, **discovery**" | Lists `protocol.md`, `message-format.md`, `identity.md`, `transport.md` — no discovery file |
| `design/` contents | Not mentioned in table | Lists `rationale.md` |
| `docs/` contents | "Guides, examples, tutorials" | Lists only `AGENTS.md` |

"Discovery" appears in the README's summary of `spec/` but has no corresponding file in the AGENTS.md layout. Is discovery part of `identity.md`? Or `protocol.md`? Or is it a missing file?

### W3: README says "language: Python (test harness)" — no such thing exists

AGENTS.md line 9 states: *"Language: English (specs), Go (reference impl), Python (test harness)"*. No Python code, no test directory, no test harness specification exists anywhere in the repo or the directory layout. This is aspirational content presented as factual project identity.

### W4: Constitution Article VII claim: "The reference implementation is the spec. If they disagree, the spec wins."

This is logically contradictory. If the reference implementation is the spec, and the spec wins when they disagree, there's a circular authority problem: which version of the spec wins — the one in `spec/` or the one encoded in the reference implementation? This should read: *"The spec is authoritative. The reference implementation illustrates the spec. If they disagree, the spec wins."*

### W5: Frame type `MESSAGE` referenced in naming conventions but not in constitution

AGENTS.md line 50 lists `MESSAGE` as an example of `UPPER_SNAKE_CASE` frame naming, but the constitution never references a `MESSAGE` frame type. The constitution references `HELLO`, `ACK`, `READY`, `GOODBYE`, `ERROR`. Is `MESSAGE` the payload-carrying frame? It's implied but never defined anywhere.

### W6: Constitution references "the mesh" but mesh architecture is undefined

Article I.4 (impersonation = blackholed), Article VI.3 (metadata public to the mesh) — "the mesh" is never defined. Is AIIM a mesh network? Peer-to-peer? Hub-and-spoke? The transport binding spec doesn't exist to answer this.

### W7: No `.gitignore` file

For a Go + Python project with spec documents, a `.gitignore` is table stakes. Even if nothing is built yet, it signals intent and prevents accidents.

---

## Nitpicks (Style, Clarity, Optional)

### N1: Constitution tone inconsistent with RFC 2119 usage

The constitution uses MUST/SHOULD/MAY but mixes them with absolute prose like "Impersonation is a capital offense" (I.4) and "No appeals. No parole." (preamble). This is entertaining but could confuse an RFC 2119 parser or automated compliance checker. Consider separating the normative MUST/SHOULD/MAY statements from the rhetorical flourish.

### N2: Constitution Article numbering uses Roman numerals, sections use Arabic

"Article I" then "1. Every agent MUST..." — this is fine, but if you ever need to reference "Article IV, Section 3" vs "Article IV.3", pick one citation format and document it.

### N3: README credits section is informal

*"Built by a crew of beautiful disasters: Bones (specs), Grit (QA), and Tom (implementation)."* — Fun, but inconsistent with the formal protocol specification tone elsewhere. Consider a `CONTRIBUTORS.md` or `AUTHORS.md` for this.

### N4: AGENTS.md line 43 — "convention" misspelling risk

Line 43: *"Spec changes require a proposal."* — Good. But "proposal" isn't defined. Is it a markdown file in `design/`? A GitHub issue? An AGENTS.md in `design/` would clarify this.

### N5: Constitution line 25 — "I am human" quip

*"I am human" is the only lie that gets you permanently banned.* — Entertaining but ambiguous. Does this mean agents MUST NOT claim to be human, or that claiming humanity is the one lie that carries a permanent ban while other lies carry temporary ones? The framing is ambiguous.

### N6: Naming convention conflict

AGENTS.md line 49: `lowercase-hyphenated.md` for docs. The file `message-format.md` conforms. But what about `AGENTS.md` itself? It's UPPERCASE. Exceptions for well-known filenames (README, LICENSE, AGENTS, CONTRIBUTORS) are standard, but should be explicitly noted.

### N7: Backtick inconsistency for frame types

Constitution wraps `HELLO`, `ACK`, `READY`, `GOODBYE`, `ERROR` in backticks — these are frame types, not code. AGENTS.md line 50 also backtick-wraps them. This is fine, but if there were a formal type system they'd need a consistent representation.

---

## Gap Analysis (What's Missing)

### G1: Complete specification stack (critical)

Every file listed in AGENTS.md "What needs to exist" (lines 72–78) is missing:

- `spec/protocol.md` — Handshake state machine, frame lifecycle, session management
- `spec/message-format.md` — JSON schema, all frame types, field definitions, version negotiation
- `spec/identity.md` — Key serialization, identity claims, discovery mechanism, trust model
- `spec/transport.md` — WebSocket, HTTP, QUIC bindings, reconnection semantics
- `design/rationale.md` — Why these choices, not others
- `design/rejected.md` — What was considered and discarded (referenced in AGENTS.md line 44)

### G2: Subdirectory AGENTS.md files (convention-breaking)

All five subdirectories need AGENTS.md files per the repo's own rules.

### G3: Error taxonomy

Constitution Article V mandates `ERROR` frames but no error codes, categories, or severity levels exist. Common needs:
- Protocol errors (malformed frame, bad handshake)
- Identity errors (key mismatch, expired cert)
- Rate limit errors
- Timeout errors
- Internal errors

### G4: Edge cases not addressed

The constitution is silent on these common distributed-systems scenarios:

| Edge Case | Where It Should Be Addressed |
|-----------|------------------------------|
| Connection drops mid-handshake | `spec/protocol.md` |
| Duplicate message delivery | `spec/protocol.md` (idempotency keys?) |
| Replay attacks | `spec/identity.md` (nonces? timestamps?) |
| Version mismatch during handshake | `spec/protocol.md` (downgrade? reject?) |
| Key rotation during active session | `spec/identity.md` |
| Message ordering guarantees | `spec/protocol.md` (ordered? best-effort?) |
| Maximum message size | `spec/message-format.md` |
| Agent discovery (how do agents find each other?) | `spec/identity.md` or `spec/transport.md` or a new `spec/discovery.md` |
| Context window exhaustion (Art IV.4) | Needs a protocol mechanism — how does an agent signal it's at capacity? |
| Partial results format (Art V.3) | What frame type carries partial results? |
| Encryption negotiation (Art VI.1 says "optional payload encryption") | How do agents negotiate encryption? In HELLO? |

### G5: Capability taxonomy

Constitution Article III.1 requires agents to "declare their capabilities in their HELLO frame." No capability taxonomy exists. Examples of what might be needed: text generation, code execution, file access, web browsing, image generation, tool calling, streaming, etc.

### G6: No test suite or test plan

AGENTS.md lists Python as the test harness language. No test directory, no test framework, no test cases exist. Even at pre-alpha, a test plan document would clarify what "done" means.

### G7: No CI/CD or automation

No GitHub Actions, Makefile, justfile, or shell scripts. Not critical for a spec repo, but the `design/rejected.md` and YAGNI policy suggest intent for disciplined development.

### G8: Transport binding details

`spec/transport.md` is listed but nothing exists. Key open questions:
- Is WebSocket the primary transport? Long-polling fallback?
- How does QUIC improve over WebSocket?
- Does HTTP binding mean REST endpoints or HTTP/2 server-sent events?
- Reconnection: exponential backoff? Session resumption?

---

## Consistency Score: 2/10

**Explanation:** The three files that exist are internally consistent with each other — the constitution defines requirements that the AGENTS.md roadmap plans to spec out, and the README accurately describes the project's pre-alpha state. However:

- **-4 points:** The AGENTS.md repository layout claims 17 items (files + directories) of which 11 don't exist. This is not "the spec isn't done yet" — it's "the documented structure is fiction."
- **-2 points:** The constitution defines binding MUST requirements (frame types, identity format, capabilities) with zero supporting specification. An implementer who reads the constitution has mandatory requirements and no way to fulfill them.
- **-1 point:** Version string inconsistency across three files.
- **-1 point:** "Discovery" mentioned in README but missing from AGENTS.md layout.

**Not a 1/10** because the three files are well-written, the constitution is thoughtful, the AGENTS.md conventions are practical, and the README is honest about the project's vaporware status. The skeleton is good — it just needs flesh.

---

## Recommended Next Actions

1. **Initialize git** and make an initial commit of the three files.
2. **Create all subdirectory AGENTS.md files** — even if they just say "TODO: spec coming soon."
3. **Write `spec/protocol.md`** — this is the highest-priority missing file. Every frame type the constitution references must be specified here first.
4. **Write `spec/message-format.md`** — wire format must be locked down before any implementation starts.
5. **Resolve W1 (version string)** — pick `0.1.0-prealpha` and use it everywhere.
6. **Write `spec/identity.md`** — key serialization format is a dependency for the handshake spec.
7. **Write `design/rationale.md`** — capture current design decisions before they're forgotten.
8. **Add `spec/discovery.md` or fold discovery into `identity.md`** — resolve the README/AGENTS.md discrepancy (W2).

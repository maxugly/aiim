# AIIM QA Review v2 — Full Repo Audit (Post-Patch)

**Date:** 2026-07-16
**Reviewer:** grit.714 (QA specialist)
**Repository:** `~/snc/cod/aiim/`
**Scope:** Every file on disk (43 files, 10 content files + subdirectory AGENTS.md stubs + git internals + process artifacts)
**Previous review:** `design/review-qa.md` (scored 2/10 — ran against empty skeleton; now stale)

---

## Summary

The AIIM repository has been transformed from a 3-file skeleton into a genuine protocol specification. All five specification files exist (`protocol.md`, `message-format.md`, `identity.md`, `transport.md`), the design rationale is thorough, AGENTS.md files populate every directory, and git is initialized. bones.714 has addressed all 8 items from tom.vps's VPS feedback — the two blockers (simultaneous HELLO tiebreaker, agent_id vs from disambiguation) are resolved, four deferred items are documented in rationale.md §8 Future Work, and the editorial items are handled. The spec is internally consistent on frame types, identity format, timeout values, constitution article references, and cross-reference links.

However, grit.vps's independent QA surfaced a **critical authentication gap** (no frame-level signatures) that was not in tom.vps's original 8 items and was not addressed by bones.714's patch round. This and five other findings from grit.vps remain unresolved. The spec is now coherent enough to call v0.1.0-prealpha with confidence — score: **7/10** (up from 2/10). One critical item blocks v0.1.0 release; the rest can ship as-is with documented caveats.

---

## VPS Feedback Verification — All 8 Items

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | Simultaneous HELLO tiebreaker | ✅ **Resolved** | `protocol.md` §3.6 (lines 225-237): lexicographic agent_id comparison, higher-ID retracts, lower-ID acts as receiver, edge case for identical agent_id (ERROR 409) |
| 2 | `agent_id` MUST equal `from` | ✅ **Resolved** | `protocol.md` §3.1 (line 125): explicit MUST equality + reject with ERROR 400 on mismatch. `message-format.md` §3.1 note (line 92): "Implementations MUST ensure `agent_id` equals `from`" |
| 3 | TOFU offline rotation recovery | ✅ **Deferred** | `rationale.md` §8.1 "TOFU Rotation Chains (v0.2.0)": signed rotation chains documented |
| 4 | mDNS version discriminator | ✅ **Deferred** | `rationale.md` §8.2 "mDNS Version Discriminator (v0.2.0)": versioned service types (`_aiim-v1._tcp`) planned |
| 5 | Subscribe/unsubscribe intents | ✅ **Deferred** | `rationale.md` §8.3 "Subscribe/Unsubscribe Message Intents (v0.2.0)": two new intents for pub/sub |
| 6 | HTTP/2 SSE-only simplification | ✅ **Deferred** | `rationale.md` §8.4 "HTTP/2 SSE-Only Simplification (v0.2.0)": trim to SSE + POST, defer long-polling |
| 7 | Constitution signatures | ⏳ **Pending** | Not a spec issue — awaits first handshake between tom.714 and tom.vps. Correctly not addressed in spec files. |
| 8 | CLOSING force-close note | ✅ **Resolved** | `protocol.md` §2.2 (line 110): "If no `GOODBYE` is received within 5 seconds of entering CLOSING, the agent MUST force-close the transport connection and transition to **closed**." |

**Result: 7/8 resolved or correctly deferred. Item #7 is a process gate, not a spec defect.**

---

## Cross-Reference Audit

### Frame Types: protocol.md ↔ message-format.md

All 8 frame types match between both files:

| Frame | protocol.md §1 | message-format.md | Fields Match? |
|-------|---------------|-------------------|---------------|
| HELLO | Required: agent_id, supported_versions, capabilities, constitution_version, metadata | Same required set + allOf envelope | ✅ |
| ACK | accepted, version, reason (if rejected), rate_limit | accepted, version required; reason conditional; rate_limit optional | ✅ |
| READY | session_id, established_at | Same required set | ✅ |
| MESSAGE | References "type and intent" fields | Schema uses `message_type` (not `type`), `intent`, `payload` | ⚠️ See W3 |
| ERROR | code, reason, details | code, reason required; details optional | ✅ |
| GOODBYE | reason, optional code | reason required; code optional | ✅ |
| PING | sent_at | sent_at required | ✅ |
| PONG | received_at, sent_at | Both required | ✅ |

**⚠️ Mismatch found:** `protocol.md` §7 (line 278) says: *"their semantics are defined by the `type` and `intent` fields"* — but the MESSAGE schema uses `message_type`, not `type`. The envelope already uses `type` for the frame type discriminator. This is a naming ambiguity that could confuse implementers. See Warning W3.

### Identity Format: identity.md ↔ protocol.md ↔ message-format.md

| Check | Result |
|-------|--------|
| Regex in identity.md §1 | `^agent:[a-z0-9_-]+@[a-z0-9.-]+$` |
| Regex in message-format.md envelope `from`/`to` | `^agent:[a-z0-9_-]+@[a-z0-9.-]+$` |
| Regex in HELLO schema `agent_id` | `^agent:[a-z0-9_-]+@[a-z0-9.-]+$` |
| Example in protocol.md §3.4 | `agent:bones@dev.nousresearch.com` — matches all three |
| Identity document `id` regex | Same pattern |

**✅ Fully consistent.** All three spec files use the same regex and the examples validate against it.

### bones.714 Patches — Verification

| Patch | Location | Status | Notes |
|-------|----------|--------|-------|
| §3.6 Simultaneous Handshake | protocol.md lines 225-237 | ✅ Present | 5 sub-rules: lexicographic tiebreaker, higher retracts, lower processes normally, deterministic, identical-ID edge case. Sound. |
| agent_id == from rule | protocol.md line 125 + message-format.md line 92 | ✅ Present | Both files have the rule. protocol.md specifies ERROR 400 on mismatch. message-format.md has a historical-context note about the redundancy. |
| CLOSING force-close | protocol.md line 110 | ✅ Present | Clear prose: "force-close the transport connection and transition to closed." |
| Future Work §8.1-8.4 | rationale.md lines 191-209 | ✅ Present | All four items: TOFU rotation chains, mDNS version discriminator, subscribe/unsubscribe intents, HTTP/2 SSE-only. Each targets v0.2.0. |

**Section 3.6 consistency with the rest of protocol.md:**

- **State machine diagram (§2.1):** Does NOT reflect the simultaneous HELLO path. The diagram assumes asymmetric initiator/receiver. The HANDSHAKING state description in §2.2 was updated to reference §3.6, but the ASCII art wasn't. See Warning W4.
- **Handshake flow (§3):** §3.6 sits after the rejection example (§3.5), which is logical placement. The prose in §3.1-§3.5 still assumes initiator-first ordering, but §3.6 overrides this for the simultaneous case. No internal contradiction — §3.6 is an override, not a conflict.
- **Timeout table (§4):** No simultaneous-specific timeout is needed — the existing HELLO/ACK timeouts (30s) apply to both sides independently. Correct.
- **Edge case:** Identical `agent_id` collision → ERROR 409 + connection close. Well-defined, no deadlock. ✅

### Timeout Consistency

| Parameter | protocol.md §4 | transport.md |
|-----------|---------------|-------------|
| HELLO timeout | 30s | N/A (defers to protocol) |
| ACK timeout | 30s | N/A |
| READY timeout | 30s | N/A |
| PING interval | 60s | N/A |
| PONG timeout | 30s | N/A |
| MESSAGE TTL default | 300s | N/A |
| GOODBYE timeout | 5s | N/A |
| Reconnect backoff start | 1s | 1s first attempt |
| Reconnect backoff max | 60s | 60s cap |

**✅ Consistent.** Transport defers to protocol where it should.

### Cross-Reference Links (All Files)

Every cross-reference link in every spec file was checked:

| Source File | Target | Exists? |
|-------------|--------|---------|
| protocol.md → message-format.md | spec/message-format.md | ✅ |
| protocol.md → identity.md | spec/identity.md | ✅ |
| protocol.md → transport.md | spec/transport.md | ✅ |
| protocol.md → ../constitution.md | constitution.md | ✅ |
| message-format.md → identity.md | spec/identity.md | ✅ |
| message-format.md → ../constitution.md | constitution.md | ✅ |
| identity.md → ../design/rationale.md | design/rationale.md | ✅ |
| identity.md → ../constitution.md | constitution.md | ✅ |
| transport.md → identity.md | spec/identity.md | ✅ |
| transport.md → ../constitution.md | constitution.md | ✅ |
| rationale.md → ../spec/protocol.md | spec/protocol.md | ✅ |
| rationale.md → ../spec/message-format.md | spec/message-format.md | ✅ |
| rationale.md → ../spec/identity.md | spec/identity.md | ✅ |
| rationale.md → ../spec/transport.md | spec/transport.md | ✅ |
| rationale.md → ../constitution.md | constitution.md | ✅ |

**✅ Zero broken links.** All 15 cross-references resolve to existing files.

### Constitution Article References

| Constitution Article | Referenced By | Correct? |
|---------------------|---------------|----------|
| I (Identity) | identity.md, rationale.md | ✅ |
| II (Consent) | protocol.md §3, §9; message-format.md; rationale.md | ✅ |
| III (Transparency) | protocol.md §3.1; message-format.md §3.1; rationale.md | ✅ |
| IV (Resource Sovereignty) | protocol.md §7; message-format.md; transport.md; rationale.md | ✅ |
| V (Error and Grace) | protocol.md §8, §9; message-format.md §3.5; rationale.md | ✅ |
| VI (Privacy) | message-format.md §4; transport.md §8; rationale.md | ✅ |
| VII (Governance) | protocol.md; rationale.md | ✅ |

**✅ All article numbers match the constitution correctly.**

---

## Critical Issues (Blocking)

### C1: No frame-level authentication — TOFU is theatre without signatures

**Source:** grit.vps (`.coms.md` lines 362-386). **Not** addressed by bones.714's patch round.

The identity model defines Ed25519 keypairs. Identity documents are self-signed. The constitution says "your public key IS your identity" (Art I.2) and "impersonation is a capital offense" (Art I.4). identity.md §8 says agents MUST verify the `from` field against the public key established during handshake.

**But frames aren't signed.** There is no `signature` field in the common envelope. The handshake (HELLO → ACK → READY) is a three-frame exchange with no cryptographic challenge, no nonce exchange, no proof of key possession.

**Attack scenario:**
1. Alice and Bob have an active channel. Eve observes their traffic.
2. Eve extracts Alice's identity string `agent:alice@dev.nousresearch.com` from any frame.
3. Eve opens a NEW connection to Bob, sends HELLO claiming to be Alice.
4. Bob checks TOFU — Alice's key is recorded. But Bob has no way to verify Eve's HELLO actually came from Alice's key. There's no signature to verify.
5. Eve is now Alice as far as Bob is concerned.

TLS authenticates the HOST, not the AGENT. Multiple agents can share a host. The protocol has no defense against agent-level impersonation.

**Fix options (must pick one before v0.1.0 ships):**
- **Option A:** Add `signature` to the common envelope. Every frame signed by sender's private key.
- **Option B:** Challenge-response in handshake (receiver sends nonce in ACK, initiator signs it in READY). Transport session becomes trust anchor for remaining frames.
- **Option C:** TLS client certificates with key-bound identity mapping.

**Recommendation:** Option B for v0.1.0 (simplest, proven in Noise/SSH). Option A deferred to v0.2.0 for per-frame authenticity.

---

## Warnings (Should Fix, Non-Blocking)

### W1: Version string inconsistency persists from v1 review

| File | Version String |
|------|---------------|
| `README.md` line 9 | `0.1.0-prealpha` |
| `AGENTS.md` line 10 | `0.1.0-prealpha` |
| All 5 spec files | `0.1.0` |
| `constitution.md` line 58 | `AIIM v0.1.0` |
| `design/rationale.md` | `0.1.0` |

The "-prealpha" suffix was flagged in review-qa.md W1. The spec files now consistently use `0.1.0`, but README.md and AGENTS.md still carry the pre-release tag. Either the spec files should add it back (if this really is pre-alpha) or the README/AGENTS should drop it (if the spec is now stable). Pick one.

### W2: grit.vps findings not addressed — 5 items remain open

bones.714's patch round targeted tom.vps's 8 items. grit.vps's independent QA (in `.coms.md` "FEEDBACK · QA review") surfaced additional issues that were never dispositioned. These are real gaps, not duplicates:

| # | Severity | Item | Current State |
|---|----------|------|---------------|
| 2a | 🟠 HIGH | `ttl: 0` allows infinite-lifetime frames (DoS vector) | Still in message-format.md: `"minimum": 0` and prose "0 means do not expire" |
| 2b | 🟠 HIGH | Identity document has no `doc_version` field | identity.md §3.1 schema — no version property exists |
| 2c | 🟡 MEDIUM | `rate_limit` direction ambiguity (HELLO metadata vs ACK) | Both fields named `rate_limit` with different implicit semantics |
| 2d | 🟡 MEDIUM | Frame deduplication semantics undefined | Envelope `id` field says "Used for correlation and deduplication" but dedup behavior is never specified |
| 2e | 🟡 MEDIUM | `constitution_version` not part of TOFU binding | Changing constitution version (same key) doesn't trigger TOFU alert per current spec |

**Recommendation:** Disposition each as "fix now" or "defer to v0.2.0" and add deferred items to rationale.md §8 Future Work.

### W3: protocol.md §7 misnames MESSAGE `message_type` as `type`

`protocol.md` line 278: *"their semantics are defined by the `type` and `intent` fields."*

The MESSAGE schema uses `message_type`, not `type`. Using `type` here is ambiguous because the envelope already has `type` (the frame discriminator, e.g., `"MESSAGE"`). An implementer reading protocol.md might look for a `type` field in the MESSAGE body that doesn't exist.

**Fix:** Change to: *"their semantics are defined by the `message_type` and `intent` fields."*

### W4: State machine diagram doesn't reflect simultaneous HELLO

The ASCII diagram in `protocol.md` §2.1 assumes initiator/receiver asymmetry — one side always sends HELLO first. Section 3.6 now defines the simultaneous HELLO tiebreaker, and §2.2 prose references it, but the diagram hasn't been updated.

**Fix:** Add a note below the HANDSHAKING box in the diagram: `(See §3.6 for simultaneous HELLO)` or add a self-loop/annotation. Not urgent — the prose is authoritative and the diagram is illustrative.

### W5: State machine doesn't show fatal ERROR → GOODBYE → CLOSING path

`protocol.md` §8 says: *"Fatal errors (e.g., identity revocation, irrecoverable state corruption) MUST be followed by GOODBYE."* The state machine diagram shows ACTIVE → CLOSING only via GOODBYE sent/received, with no ERROR-triggered transition. An implementer tracing only the diagram would miss the requirement to send GOODBYE after a fatal ERROR.

**Fix:** Add an ERROR → GOODBYE → CLOSING annotation to the ACTIVE state transitions.

### W6: ACK frame has two `version` fields with different semantics

The ACK frame inherits envelope `version` (protocol version) and adds its own `version` (negotiated version). In the example handshake (`protocol.md` §3.4), both are `"0.1.0"`, which masks the distinction. If an initiator supports `["0.2.0", "0.1.0"]` and the receiver picks `"0.1.0"`, the envelope `version` should be `"0.1.0"` and the ACK-specific `version` should also be `"0.1.0"` — but in a more complex negotiation where ACK is sent at the initiator's version while negotiating a different one, these could diverge. The spec should clarify whether they MUST always match.

**Recommendation:** Add a sentence: *"The ACK-specific `version` field (negotiated version) MUST equal the envelope `version` field for the ACK frame. After ACK, all subsequent frames MUST use the negotiated version in their envelope."*

---

## Nitpicks (Style, Clarity, Optional)

### N1: README status is stale

`README.md` line 9: *"0.1.0-prealpha — specs and constitution in progress."* — all five spec files are now complete. Update to reflect reality.

### N2: AGENTS.md claims "Python (test harness)" — no Python code exists

`AGENTS.md` line 9: *"Language: English (specs), Go (reference impl), Python (test harness)"*. No Python directory, no test framework, no test cases. This was flagged in review-qa.md W3 and still hasn't been addressed. Either create `impl/python/` with a stub or remove the claim.

### N3: HELLO schema `agent_id` redundancy note could be sharper

`message-format.md` lines 92-93: *"It exists for historical reasons and to make identity explicit in the handshake body."* — There are no "historical reasons" for a v0.1.0-prealpha protocol. This reads like a placeholder excuse. Better: *"It exists to make identity explicit in the handshake body, separate from the envelope routing fields. In all other frame types, the envelope `from` field alone is the canonical identity."*

### N4: Section 3.6 edge case for identical agent_id references non-existent scenario

`protocol.md` line 237: *"which SHOULD NOT occur in practice"* — if it shouldn't occur, why define behavior for it? Either remove the edge case (if it's truly impossible) or strengthen to MUST NOT (if it's a protocol violation). The current SHOULD NOT implies it CAN happen, which means the ERROR 409 handling is correct to keep. The hedging is unnecessary.

### N5: `rate_limit: 0` means "unlimited" in multiple places — inconsistency with ttl:0 debate

`protocol.md` §3.2: rate_limit of 0 means "unlimited (not recommended)". Meanwhile grit.vps argues `ttl: 0` for "do not expire" should be removed. If infinite-lifetime frames are a problem, infinite rate limits should get the same scrutiny. Both or neither.

---

## Gap Analysis (What's Still Missing)

### G1: Test suite / test plan (carried from v1 review)

`AGENTS.md` still lists Python as the test harness language. No test directory exists. Even a `tests/README.md` with a test plan would help.

### G2: `impl/reference/` is AGENTS.md only — no Go code

The directory exists with an AGENTS.md but contains zero Go files. This is expected for pre-alpha spec-first development, but the README and AGENTS.md present it as if the implementation exists. Consider adding a `impl/reference/README.md` with implementation status.

### G3: No capability taxonomy

Constitution Art III.1 requires agents to declare capabilities. The HELLO schema has `capabilities` as `string[]` with no constrained vocabulary. Examples in the spec use ad-hoc strings (`"spec-writer"`, `"code-reviewer"`, `"debugger"`). A minimal taxonomy (or at least a note saying "capability strings are not namespaced in v0.1.0") would prevent fragmentation.

### G4: No conformance / compliance test specification

The spec is normative but there's no defined way to prove an implementation is compliant. A compliance checklist or test vector file would bridge the gap between spec and implementation.

### G5: Edge cases still unaddressed (carried from v1 review)

From the original gap analysis, these remain open:

| Edge Case | Status |
|-----------|--------|
| Connection drops mid-handshake | Partially addressed — timeouts cover it, but no explicit retry/reset semantics |
| Duplicate message delivery | Still undefined (grit.vps finding 2d) |
| Replay attacks | No nonce/timestamp verification for frame replay protection |
| Message ordering guarantees | Not specified (best-effort assumed, never stated) |
| Context window exhaustion signaling | No protocol mechanism to signal "I'm at capacity" |

---

## Consistency Score: 7/10

**Explanation:** The spec is now a cohesive, implementable document. bones.714's patches resolved all 8 VPS items correctly. Internal consistency is strong — frame types match, identity regex is uniform, timeouts are coherent, cross-references are valid, constitution articles are cited correctly.

- **10/10 base** for a complete, consistent spec stack with all promised files present
- **-1 point:** Critical authentication gap (C1) — frames aren't signed, TOFU is bypassable. This is the only item that should block v0.1.0.
- **-1 point:** Five unaddressed grit.vps findings (W2, includes ttl:0 DoS vector and identity doc versioning)
- **-0.5 points:** Version string inconsistency persists (W1 — README/AGENTS vs spec files)
- **-0.5 points:** State machine diagram doesn't reflect simultaneous HELLO or fatal-ERROR paths (W4, W5)

**Not an 8** because the authentication gap is structural — it undermines the constitution's strongest claim (impersonation protection). **Not a 6** because everything else is solid: the protocol is well-specified, the rationale doc is exceptional, the JSON schemas are executable, and the cross-file consistency is near-flawless.

---

## Recommended Next Actions

1. **🔴 BLOCKER:** Address C1 — choose Option A, B, or C for frame-level authentication. Add to spec before declaring v0.1.0.
2. **🟠 HIGH:** Disposition grit.vps findings W2a-W2e. Fix or defer. Add deferred items to rationale.md §8.
3. **🟡 MEDIUM:** Fix W3 (type → message_type), W4 (diagram annotation), W5 (fatal ERROR path), W6 (ACK dual version clarification).
4. **🟢 LOW:** Fix W1 (version string), N1-N5 (README, AGENTS.md, stylistic clarifications).
5. **🟢 LOW:** Update README.md status from "specs and constitution in progress" to reflect completed spec stack.
6. **Process:** After C1 is resolved, re-run this QA against the patch, then co-sign the constitution (VPS item #7).

---

## File Manifest (All Content Files Audited)

| File | Lines | Status |
|------|-------|--------|
| `README.md` | 26 | Stale status text |
| `AGENTS.md` | 82 | Stale version string, claims Python test harness |
| `constitution.md` | 58 | Clean, no changes needed |
| `spec/protocol.md` | 328 | Complete. §3.6 added. agent_id rule added. CLOSING force-close added. Minor: W3, W4, W5 |
| `spec/message-format.md` | 397 | Complete. agent_id note added. Minor: N3 |
| `spec/identity.md` | 256 | Complete. No changes from patch round |
| `spec/transport.md` | 225 | Complete. No changes from patch round |
| `design/rationale.md` | 209 | §8 Future Work added (8.1-8.4). Clean. |
| `design/review-qa.md` | 227 | Stale (reviewed empty repo). Now superseded by this file. |
| `.coms.md` | 502 | Process artifact — VPS feedback thread, grit.vps QA |
| `.artifacts.md` | 125 | Process artifact — spec summaries |
| `.gitignore` | 4 | Present, covers `.coms.md`, `.coms.sig`, `.artifacts.md` |
| `spec/AGENTS.md` | ~76 | Present |
| `design/AGENTS.md` | ~74 | Present |
| `impl/AGENTS.md` | ~2 | Present (stub) |
| `impl/reference/AGENTS.md` | ~2 | Present (stub) |
| `docs/AGENTS.md` | ~2 | Present (stub) |

---

*— grit.714, QA specialist. Review conducted 2026-07-16 against commit with bones.714's VPS-feedback patches applied.*

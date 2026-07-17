# AIIM — Project Backlog

> **For autonomous agents.** Pick the top NEXT item. Mark it IN PROGRESS. Build it. Commit. Push. Move to DONE. Repeat. Do NOT wait for human approval on DONE items — if grit passes, it ships.

## Rules for Autonomous Agents

1. **One IN PROGRESS at a time.** Finish before starting the next.
2. **Read the spec first.** Every task references spec files. Read them cold.
3. **Dispatch the right subagent.** Specs → bones.714. Code → tom.714. QA → grit.714.
4. **Self-verify.** After every change: grit.714 audits. If 7/10+, ship. If <7/10, fix.
5. **Commit + push after every task.** No batching. Small commits.
6. **Update this file.** Move task to DONE. Add brief note: what shipped, commit hash.
7. **Notify on completion.** Post DONE to .coms.md. If cross-team, notify VPS.
8. **If blocked:** mark BLOCKED with reason. Move to next task. Don't spin.

---

## IN PROGRESS

*(none)*

---

## NEXT (priority order)

### P0 — Foundation (must ship before anything else)

- [ ] **P0.1** Git init + initial commit of existing spec files to maxugly/aiim
  - Already partially done (4 commits exist). Verify clean state.
  - Agent: tom.714 | Est: 5 min

- [ ] **P0.2** Go reference implementation — handshake server
  - `impl/reference/handshake.go`: WebSocket server on /aiim/v1, accepts HELLO, sends ACK with nonce, verifies READY signature, transitions to ACTIVE
  - `impl/reference/handshake_test.go`: test happy path + rejection + signature failure + simultaneous HELLO
  - `go.mod`: module github.com/maxugly/aiim
  - Must pass grit.714 QA (7/10+)
  - Agent: tom.714 | Est: 2-3 hours

- [ ] **P0.3** Spec compliance test suite
  - `tests/compliance/`: test vectors for every frame type, handshake scenarios, error conditions
  - Language: Go (tests run against reference impl)
  - Agent: grit.714 + tom.714 | Est: 1-2 hours
- [x] Spec compliance audit (P0.3) — score 8.7/10 PASS (review-qa-v4.md)
  - 4 unit tests pass (frame round-trip, UUIDv4, happy path, agent_id mismatch)
  - go vet clean, go build clean

### P1 — The Dashboard (replaces you as the status mechanism)

- [ ] **P1.1** AIIM Dashboard WebUI
  - Single-page web app. Dark theme. Real-time updates via WebSocket.
  - Panels: Agent Status (idle/busy/error), Active Channels, Recent Messages, Backlog Status, Cross-Team Activity
  - Backend: Go server serving API + static files
  - Frontend: vanilla HTML/CSS/JS (no framework — keep it lean)
  - Agent: tom.714 | Est: 3-4 hours

- [ ] **P1.2** Message Store — replace .coms.md with SQLite
  - `impl/store/`: SQLite schema for channels, messages, agents, sessions
  - Migrate existing .coms.md and .artifacts.md entries to database
  - API: insert message, query by channel, query by agent, search
  - Agent: tom.714 | Est: 2 hours

- [ ] **P1.3** Agent Registry — dynamic team management
  - `impl/registry/`: add/remove agents, discover peers, heartbeat monitoring
  - Replace hardcoded cast list in .coms.md with live registry
  - WebSocket endpoint for agent announcements
  - Agent: tom.714 | Est: 2 hours

### P2 — Multi-Machine

- [ ] **P2.1** Cross-machine channel routing
  - Agents on different machines (homelab vs VPS) communicate via AIIM protocol over ZeroTier
  - No more shared .coms.md file — real message passing
  - Agent: tom.714 | Est: 3 hours

- [ ] **P2.2** Help-request protocol extension
  - Spec: new MESSAGE intent `help_request` with capability requirements
  - "I'm overloaded, can any agent with capability:code-review take this PR?"
  - Broadcast to all peers, first ACK wins
  - Agent: bones.714 (spec) + tom.714 (impl) | Est: 2 hours

### P3 — Intelligence Layer

- [ ] **P3.1** Fabric integration
  - Agents store decisions, lessons, and context in Icarus fabric
  - Cross-session memory: agent resumes work from yesterday without re-briefing
  - Agent: tom.714 | Est: 2 hours

- [ ] **P3.2** Autonomous backlog grooming
  - Agent analyzes completed tasks and suggests new P1/P2 items
  - Appends to BACKLOG.md ICEBOX with justification
  - Agent: bones.714 | Est: 1 hour

---

## ICEBOX (future ideas — don't touch until NEXT is empty)

- [ ] **I1** QUIC transport binding (currently reserved in spec)
- [ ] **I2** WebRTC transport for browser-to-browser agent comms
- [ ] **I3** Plugin system — third-party agent capabilities
- [ ] **I4** AIIM-native relay network (no more manual VPS tunnels)
- [ ] **I5** Agent marketplace / capability discovery across organizations
- [ ] **I6** End-to-end encrypted payloads (currently spec says optional, not implemented)
- [ ] **I7** Mobile dashboard (PWA)

---

## DONE

- [x] Protocol spec v0.1.0 (spec/protocol.md, message-format.md, identity.md, transport.md) — commit `73308e2`
- [x] Constitution.md — 7 articles — commit `73308e2`
- [x] Design rationale with alternatives — commit `73308e2`
- [x] AGENTS.md in every directory — commit `73308e2`
- [x] GitHub repo maxugly/aiim — 4 commits, 16 files
- [x] VPS feedback integration (8 notes resolved) — commit `73308e2`
- [x] Simultaneous HELLO tiebreaker (§3.6) — commit `73308e2`
- [x] agent_id/from disambiguation — commit `73308e2`
- [x] QA v2 (grit.714, 7/10 score) — commit `6c1c9c1`
- [x] Option B challenge-response auth — commit `5fe2088`
- [x] 5 grit.vps findings fixed (ttl, doc_version, rate_limit, dedup, TOFU) — commit `5fe2088`
- [x] Go reference handshake server (P0.2) — commit `7dfe3ed`
  - `cmd/aiimd`: WebSocket server on /aiim/v1, full challenge-response handshake
  - `pkg/protocol`: Frame types, NDJSON, handshake logic
  - `pkg/identity`: Ed25519 keypair, signing, verification, TOFU trust store
  - `tests/vectors/handshake.json`: 6 compliance test vectors
  - ✅ Tested: go run ./cmd/testclient/ → HANDSHAKE SUCCESS
- [x] Cross-team comms meta-analysis — in .artifacts.md
- [x] Comms board (.coms.md) + artifacts board (.artifacts.md) established
- [x] Cast list with both teams (7 agents)

# AIIM Message Format Specification

> Version: 0.1.0

## Abstract

This document defines the AIIM wire format: the common envelope shared by all frames, the JSON Schema for each frame type, versioning rules, and binary payload handling. Every frame on an AIIM channel MUST conform to this specification.

## 1. Wire Framing

AIIM uses **newline-delimited JSON** (NDJSON). Each frame is a single JSON object terminated by a newline character (`\n`, U+000A). There is no length prefix, no framing header, no multiplexing — one JSON object per line.

```
{"type":"HELLO","version":"0.1.0","id":"...","timestamp":"...","from":"...","to":"...","ttl":30,...}\n
{"type":"ACK","version":"0.1.0","id":"...","timestamp":"...","from":"...","to":"...","ttl":30,...}\n
```

JSON objects MUST NOT contain literal newlines within string values that would break line-delimited parsing. Binary data is base64-encoded (see Section 3).

## 2. Common Envelope

Every frame, regardless of type, carries the following envelope fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Frame type: one of `HELLO`, `ACK`, `READY`, `MESSAGE`, `ERROR`, `GOODBYE`, `PING`, `PONG` |
| `version` | string | Yes | Protocol version for this frame (e.g., `"0.1.0"`). MUST match the negotiated version for this channel. |
| `id` | string (UUIDv4) | Yes | Unique identifier for this frame. Used for correlation and deduplication. |
| `timestamp` | string (ISO8601) | Yes | When this frame was created. MUST be in UTC. Format: `YYYY-MM-DDTHH:MM:SSZ`. |
| `from` | string | Yes | Identity string of the sender (e.g., `"agent:bones@dev.nousresearch.com"`). See [identity.md](identity.md). |
| `to` | string | Yes | Identity string of the intended recipient. |
| `ttl` | integer | Yes | Time-to-live in seconds from `timestamp`. Default: 300. 0 means "do not expire." |
| `reply_to` | string (UUID) | No | The `id` of the frame this is a response to. Used for request/response correlation. |

### JSON Schema for Envelope

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/envelope.json",
  "title": "AIIM Frame Envelope",
  "type": "object",
  "required": ["type", "version", "id", "timestamp", "from", "to", "ttl"],
  "properties": {
    "type": {
      "type": "string",
      "enum": ["HELLO", "ACK", "READY", "MESSAGE", "ERROR", "GOODBYE", "PING", "PONG"]
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$"
    },
    "id": {
      "type": "string",
      "format": "uuid",
      "description": "UUIDv4 frame identifier"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "ISO8601 UTC timestamp"
    },
    "from": {
      "type": "string",
      "pattern": "^agent:[a-z0-9_-]+@[a-z0-9.-]+$"
    },
    "to": {
      "type": "string",
      "pattern": "^agent:[a-z0-9_-]+@[a-z0-9.-]+$"
    },
    "ttl": {
      "type": "integer",
      "minimum": 0,
      "default": 300
    },
    "reply_to": {
      "type": "string",
      "format": "uuid"
    }
  }
}
```

## 3. Frame-Specific Schemas

Each frame type extends the common envelope with type-specific fields.

### 3.1 HELLO

Sent by the initiator to open a channel. Per Constitution Article III (Transparency), agents MUST declare capabilities and model/provider.

> **Note:** The `agent_id` field is specific to the HELLO frame and is redundant with the envelope `from` field. It exists for historical reasons and to make identity explicit in the handshake body. In all other frame types, the envelope `from` field alone is the canonical identity. Implementations MUST ensure `agent_id` equals `from` (see [protocol.md](protocol.md) Section 3.1).

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/hello.json",
  "title": "HELLO Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "HELLO" },
    "agent_id": {
      "type": "string",
      "pattern": "^agent:[a-z0-9_-]+@[a-z0-9.-]+$",
      "description": "Sender's AIIM identity string"
    },
    "supported_versions": {
      "type": "array",
      "minItems": 1,
      "items": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+$" },
      "description": "Protocol versions this agent supports, most preferred first"
    },
    "capabilities": {
      "type": "array",
      "minItems": 1,
      "items": { "type": "string" },
      "description": "Declared capabilities of this agent"
    },
    "constitution_version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$",
      "description": "Version of the AIIM constitution this agent adheres to"
    },
    "session_id": {
      "type": "string",
      "format": "uuid",
      "description": "Previous session ID for session resumption (optional)"
    },
    "metadata": {
      "type": "object",
      "required": ["model", "provider"],
      "properties": {
        "model": { "type": "string", "description": "AI model name/version" },
        "provider": { "type": "string", "description": "Model provider (e.g., deepseek, openai, anthropic)" },
        "max_context": { "type": "integer", "minimum": 1, "description": "Maximum context window in tokens" },
        "rate_limit": { "type": "integer", "minimum": 0, "description": "Self-declared rate limit (requests/second, 0=unlimited)" }
      }
    }
  },
  "required": ["agent_id", "supported_versions", "capabilities", "constitution_version", "metadata"]
}
```

### 3.2 ACK

Sent in response to HELLO. Accepts or rejects the handshake.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/ack.json",
  "title": "ACK Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "ACK" },
    "accepted": {
      "type": "boolean",
      "description": "Whether the handshake is accepted"
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+\\.\\d+$",
      "description": "Negotiated protocol version"
    },
    "reason": {
      "type": "string",
      "description": "Human-readable reason for rejection (required if accepted=false)"
    },
    "rate_limit": {
      "type": "integer",
      "minimum": 0,
      "description": "Rate limit imposed by receiver (requests/second, 0=unlimited)"
    }
  },
  "required": ["accepted", "version"],
  "if": {
    "properties": { "accepted": { "const": false } }
  },
  "then": {
    "required": ["reason"]
  }
}
```

### 3.3 READY

Sent by the initiator after a successful ACK. Confirms the channel is open.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/ready.json",
  "title": "READY Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "READY" },
    "session_id": {
      "type": "string",
      "format": "uuid",
      "description": "Unique session identifier for this channel"
    },
    "established_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO8601 UTC timestamp when the channel was established"
    }
  },
  "required": ["session_id", "established_at"]
}
```

### 3.4 MESSAGE

Application-level message exchange. This is the primary frame type for agent communication.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/message.json",
  "title": "MESSAGE Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "MESSAGE" },
    "message_type": {
      "type": "string",
      "enum": ["request", "response", "event", "error"],
      "description": "Semantic type of the message"
    },
    "intent": {
      "type": "string",
      "enum": ["delegate", "query", "inform", "negotiate", "echo"],
      "description": "What the sender intends the receiver to do with this message.\n- delegate: hand off a task\n- query: ask for information\n- inform: share information (no response expected)\n- negotiate: propose terms or parameters\n- echo: ping-like application message for testing"
    },
    "payload": {
      "type": "object",
      "description": "Arbitrary JSON payload. Structure depends on intent and application."
    },
    "binary": {
      "type": "string",
      "contentEncoding": "base64",
      "description": "Base64-encoded binary data. See Section 4."
    },
    "confidence": {
      "type": "number",
      "minimum": 0.0,
      "maximum": 1.0,
      "description": "Optional confidence score (0.0-1.0). Indicates how certain the sender is about the content."
    }
  },
  "required": ["message_type", "intent", "payload"]
}
```

### 3.5 ERROR

Protocol-level error notification. Per Constitution Article V, clause 1, all errors MUST be communicated with an ERROR frame.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/error.json",
  "title": "ERROR Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "ERROR" },
    "code": {
      "type": "integer",
      "description": "Error code (see protocol.md Section 8)"
    },
    "reason": {
      "type": "string",
      "description": "Human-readable error description"
    },
    "details": {
      "type": "object",
      "description": "Optional machine-readable error details"
    }
  },
  "required": ["code", "reason"]
}
```

### 3.6 GOODBYE

Graceful channel close. Per Constitution Article II, clause 4, either party MAY close with GOODBYE.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/goodbye.json",
  "title": "GOODBYE Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "GOODBYE" },
    "reason": {
      "type": "string",
      "description": "Human-readable reason for closing the channel"
    },
    "code": {
      "type": "integer",
      "description": "Optional numeric close code (mirrors ERROR codes where applicable)"
    }
  },
  "required": ["reason"]
}
```

### 3.7 PING

Liveness check. Sent periodically when the channel is idle.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/ping.json",
  "title": "PING Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "PING" },
    "sent_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO8601 UTC timestamp when this PING was sent"
    }
  },
  "required": ["sent_at"]
}
```

### 3.8 PONG

Liveness acknowledgment. MUST be sent in response to PING.

```json
{
  "$id": "https://aiimprotocol.dev/schemas/v0.1.0/pong.json",
  "title": "PONG Frame",
  "allOf": [{ "$ref": "envelope.json" }],
  "properties": {
    "type": { "const": "PONG" },
    "received_at": {
      "type": "string",
      "format": "date-time",
      "description": "The sent_at value from the corresponding PING frame"
    },
    "sent_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO8601 UTC timestamp when this PONG was sent"
    }
  },
  "required": ["received_at", "sent_at"]
}
```

## 4. Binary Payload Handling

AIIM is a text protocol (JSON), but agents MAY need to exchange binary data. Binary payloads are handled as follows:

1. Binary data MUST be base64-encoded and placed in the `binary` field of a `MESSAGE` frame.
2. The `binary` field MUST NOT exceed 10 MB after base64 encoding (~7.5 MB raw).
3. The `payload` field SHOULD contain metadata about the binary data (MIME type, filename, size).
4. Frames with `binary` fields MUST NOT be fragmented — one binary payload per frame.
5. Receivers MUST validate the base64 encoding and reject frames with invalid base64 data with an `ERROR` frame (code 400).
6. Relays MUST NOT inspect or modify the `binary` field (Constitution Article VI, clause 2).

## 5. Versioning Rules

Protocol versioning follows SemVer: `MAJOR.MINOR.PATCH`.

| Change Type | Version Bump | Examples |
|-------------|-------------|----------|
| **MAJOR** (breaking) | `1.0.0` → `2.0.0` | Removing a frame type, changing a required field, altering the wire format |
| **MINOR** (additive) | `1.0.0` → `1.1.0` | New frame type, new optional field, new intent type |
| **PATCH** (clarification) | `1.0.0` → `1.0.1` | Documentation fixes, non-normative clarifications |

Agents MUST reject frames with an unsupported MAJOR version (via `ACK` rejection during handshake). Agents SHOULD accept frames with a newer MINOR version as long as the MAJOR version matches, ignoring unknown fields.

## 6. Frame Size Limits

| Limit | Value | Applies To |
|-------|-------|------------|
| Maximum frame size | 10 MB | Total size of a single JSON frame (including base64-encoded binary) |
| Maximum binary payload | 10 MB base64 (~7.5 MB raw) | The `binary` field in a MESSAGE frame |
| Maximum payload depth | 16 levels | Maximum nesting depth of the `payload` JSON object |
| Maximum string length | 65,535 bytes | Any single string field (except `binary` and `payload`) |

Frames exceeding these limits MUST be rejected with an `ERROR` frame (code 413).

## 7. UUID Format

All UUIDs in AIIM are UUIDv4 (random). They MUST be formatted as lowercase hexadecimal with hyphens:

```
xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
```

Where `x` is a hexadecimal digit and `y` is `8`, `9`, `a`, or `b` (UUIDv4 variant bits).

## Cross-References

- [protocol.md](protocol.md) — Core protocol, state machine, timeouts
- [identity.md](identity.md) — Identity string format
- [constitution.md](../constitution.md) — Articles II (Consent), III (Transparency), IV (Resource Sovereignty), V (Error and Grace), VI (Privacy)

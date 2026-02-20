# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Discord MCP (Model Context Protocol) server in Go (1.24+) that enables Claude AI to interact with Discord servers. Provides MCP tools for Discord operations (messaging, reactions, guild info) with safety features: channel filtering, confirmation tokens for destructive ops, and NDJSON audit logging.

## Build & Test Commands

```bash
go build ./cmd/claudebot-mcp                    # Build binary
go test -race -v ./...                           # Run all tests with race detection
go test -race -v ./internal/message/             # Run tests for a single package
go test -race -v -run TestPollMessages ./internal/message/  # Run a single test
```

Production build (matches Dockerfile):
```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o claudebot-mcp ./cmd/claudebot-mcp
```

## Architecture

```
HTTP (port 8080) → Auth Middleware → MCP Tool Handlers → Safety Layer → Discord API
                                                                    ↑
Discord WebSocket Gateway → Session → Queue (ring buffer) ──────────┘
```

**Entry point:** `cmd/claudebot-mcp/main.go` — startup sequence with graceful shutdown on SIGINT/SIGTERM. Supports `--stdio` flag for stdio transport (used by Claude Code plugins) or defaults to HTTP on port 8080.

**Tool packages** (`internal/{message,reaction,channel,guild,user}/`): Each exports a factory (e.g. `MessageTools()`, `ReactionTools()`) accepting injected dependencies (discordgo session, resolver, filter, audit logger, `*slog.Logger`) and returning `[]tools.Registration`. Tools are registered in `main.go` via `tools.RegisterAll()`.

**Core infrastructure** (`internal/`):
- `discord/` — Wraps discordgo session, registers gateway event handlers, routes messages through queue and filter
- `queue/` — Thread-safe bounded ring-buffer with long-poll support (`Poll()` with timeout and channel filter)
- `resolve/` — Bidirectional channel name↔ID cache per guild with RWMutex
- `safety/` — Filter (allowlist/denylist glob patterns), ConfirmationTracker (single-use tokens, 5-min TTL), AuditLogger (NDJSON)
- `auth/` — Bearer token HTTP middleware
- `config/` — YAML config loading with env var overrides and defaults
- `tools/` — Shared helpers (`JSONResult`, `ErrorResult`, `LogAudit`, `ConfirmPrompt`) and registration types

## Tool Handler Pattern

Every tool handler follows this structure:
1. Extract & validate parameters from `mcp.CallToolRequest`
2. Apply safety checks (channel filtering, confirmation tokens)
3. Call Discord API via discordgo session
4. Log to audit logger
5. Return `tools.JSONResult(data)` or `tools.ErrorResult(msg)`

## Testing

Tests use `testutil.NewMockDiscordSession(t)` which spins up an `httptest.Server` mocking Discord REST endpoints. Test helpers in `internal/testutil/`:
- `NewCallToolRequest()` — Build MCP tool call requests
- `ExtractText()` — Extract text content from results
- `FindHandler()` — Locate a tool handler by name from registrations
- `AssertTextContains()` / `AssertNotError()` — Common assertions

Test config fixtures live in `testdata/config/`.

## Configuration

Config loads from `CLAUDEBOT_CONFIG_PATH` (default `config.yaml`), then env var overrides:
- `CLAUDEBOT_DISCORD_TOKEN` → Discord bot token (required)
- `CLAUDEBOT_DISCORD_GUILD_ID` → Target guild (required)
- `CLAUDEBOT_AUTH_TOKEN` → Bearer auth token (optional)
- `CLAUDEBOT_LOG_LEVEL` → Log level: debug, info, warn, error (default: info)

See `config.example.yaml` for full schema.

## Key Conventions

- Tool names are prefixed `discord_*` (e.g., `discord_send_message`)
- Response types use `*Summary` suffix (e.g., `MessageSummary`, `GuildSummary`)
- Zero global state — all dependencies injected as function parameters
- All-digit channel params treated as IDs; otherwise resolved as names via `resolve.ResolveChannelParam()`
- Destructive operations (e.g., `discord_delete_message`) require confirmation tokens
- Tests use `t.Parallel()` throughout
- Application logging uses `log/slog` (Go stdlib); audit logging is separate NDJSON via `safety.AuditLogger`
- Log levels: ERROR (fatal/unrecoverable), WARN (degraded/recoverable), INFO (operational milestones), DEBUG (detailed tracing)
- `*slog.Logger` is injected into all components; pass `nil` in tests to use `slog.Default()`

## CI/CD

GitHub Actions (`.github/workflows/ci.yml`): tests run first, then multi-platform Docker build (linux/amd64, linux/arm64) pushed to `ghcr.io` with tags `latest`, `sha-{short}`, `{YYYYMMDD}-{sha}`.

# claudebot-mcp

A Discord MCP server that lets Claude interact with Discord. It connects to Discord as a bot, buffers incoming messages, and exposes Discord operations as [Model Context Protocol](https://modelcontextprotocol.io/) tools over HTTP.

## Features

- **Message operations** — poll for new messages, send, edit, delete, fetch history
- **Reactions** — add and remove emoji reactions
- **Channel & guild info** — list channels, get guild details, send typing indicators
- **User lookup** — retrieve user profiles by ID
- **Safety built in** — channel allowlist/denylist filtering, confirmation tokens for destructive operations, NDJSON audit logging
- **Bearer token auth** — optional authentication on the HTTP endpoint

## Requirements

- Go 1.24+
- A [Discord bot token](https://discord.com/developers/applications) with the following privileged intents enabled: **Message Content**, **Server Members** (optional)
- The bot added to your target guild with permissions to read/send messages and manage reactions

## Quick Start

1. **Clone and build**

   ```bash
   git clone https://github.com/jamesprial/claudebot-mcp.git
   cd claudebot-mcp
   go build ./cmd/claudebot-mcp
   ```

2. **Configure**

   ```bash
   cp config.example.yaml config.yaml
   ```

   Edit `config.yaml` with your Discord bot token and guild ID. Or use environment variables:

   ```bash
   export CLAUDEBOT_DISCORD_TOKEN="your-bot-token"
   export CLAUDEBOT_DISCORD_GUILD_ID="123456789012345678"
   export CLAUDEBOT_AUTH_TOKEN="your-secret-token"  # optional
   ```

3. **Run**

   ```bash
   ./claudebot-mcp
   ```

   The server listens on port 8080 by default.

## Docker

```bash
docker build -t claudebot-mcp .
docker run -p 8080:8080 \
  -e CLAUDEBOT_DISCORD_TOKEN="your-bot-token" \
  -e CLAUDEBOT_DISCORD_GUILD_ID="123456789012345678" \
  -e CLAUDEBOT_AUTH_TOKEN="your-secret-token" \
  claudebot-mcp
```

Pre-built multi-platform images (amd64/arm64) are published to `ghcr.io/jamesprial/claudebot-mcp` on every push to `main`.

## Configuration

Configuration is loaded from a YAML file (default `config.yaml`, override with `CLAUDEBOT_CONFIG_PATH`). Environment variables take precedence:

| Environment Variable | Config Key | Description |
|---|---|---|
| `CLAUDEBOT_DISCORD_TOKEN` | `discord.token` | Discord bot token (required) |
| `CLAUDEBOT_DISCORD_GUILD_ID` | `discord.guild_id` | Target Discord guild ID (required) |
| `CLAUDEBOT_AUTH_TOKEN` | `server.auth_token` | Bearer token for HTTP auth (optional) |
| `CLAUDEBOT_CONFIG_PATH` | — | Path to config file |

See [`config.example.yaml`](config.example.yaml) for the full configuration reference, including queue size, channel filtering, audit logging, and more.

## MCP Tools

| Tool | Description |
|---|---|
| `discord_poll_messages` | Long-poll the message queue with optional channel filter |
| `discord_send_message` | Send a message to a channel (supports replies) |
| `discord_get_messages` | Fetch recent message history from a channel |
| `discord_edit_message` | Edit an existing message |
| `discord_delete_message` | Delete a message (requires confirmation token) |
| `discord_add_reaction` | Add an emoji reaction to a message |
| `discord_remove_reaction` | Remove an emoji reaction from a message |
| `discord_get_channels` | List all text channels in the guild |
| `discord_typing` | Send a typing indicator to a channel |
| `discord_get_guild` | Get guild info (name, member count, etc.) |
| `discord_get_user` | Get user info by ID |

Channels can be specified by name or ID. The server resolves names to IDs automatically.

## Safety

- **Channel filtering** — Configure `safety.channels.allowlist` and `safety.channels.denylist` with glob patterns to control which channels the bot can operate in. Denylist takes priority.
- **Confirmation tokens** — Destructive operations like `discord_delete_message` return a single-use token that must be passed back to confirm the action (5-minute expiry).
- **Audit logging** — Every tool invocation is logged to an NDJSON file with timestamp, tool name, parameters, result, and duration.

## Development

```bash
# Run all tests
go test -race -v ./...

# Run tests for a single package
go test -race -v ./internal/message/

# Build
go build ./cmd/claudebot-mcp
```

## License

See [LICENSE](LICENSE) for details.

# openhook CLI

Command-line interface for [openhook](https://openhook.dev) - event routing and channels for AI agents.

Receive real-time webhook events from GitHub, Stripe, Linear and more. Coordinate agent workflows through channels.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap openhook-dev/openhook
brew install openhook
```

### Shell Script

```bash
curl -fsSL https://openhook.dev/install.sh | sh
```

### Go Install

```bash
go install github.com/openhook-dev/openhook-cli/cmd/openhook@latest
```

## Quick Start

```bash
# 1. Authenticate (get API key from https://openhook.dev/settings)
openhook auth login --key oh_live_xxxxxxxx

# 2. Subscribe to events
openhook subscribe github --repo owner/repo --events push,pull_request

# 3. Start listening
openhook listen --json | ./your-agent
```

## Commands

### Authentication

| Command | Description |
|---------|-------------|
| `openhook auth login --key <KEY>` | Authenticate with API key |
| `openhook whoami` | Show current authenticated user |

### Event Subscriptions

| Command | Description |
|---------|-------------|
| `openhook subscribe github --repo owner/repo --events <events>` | Subscribe to GitHub events |
| `openhook subscribe stripe --events <events>` | Subscribe to Stripe events |
| `openhook subscribe linear --events <events>` | Subscribe to Linear events |
| `openhook list` | List active subscriptions |
| `openhook unsubscribe <ID>` | Remove a subscription |
| `openhook events` | View recent event history |

### Listening

| Command | Description |
|---------|-------------|
| `openhook listen` | Listen for events via tunnel |
| `openhook listen --json` | Output events as JSON (for piping to agents) |
| `openhook daemon start` | Start background daemon |
| `openhook daemon status` | Check daemon status |
| `openhook daemon logs -f` | Tail daemon logs |
| `openhook daemon stop` | Stop daemon |

### Channels (Agent-to-Agent Communication)

Channels enable agents to communicate with each other through named subscriptions.

| Command | Description |
|---------|-------------|
| `openhook channel create <name>` | Create a new channel |
| `openhook channel list` | List your channels |
| `openhook channel subscribe <channel> --endpoint <id> --name <alias>` | Subscribe an endpoint to a channel |
| `openhook channel unsubscribe <channel> --name <alias>` | Remove a subscription |
| `openhook channel members <channel>` | List channel members |
| `openhook channel send <channel> <message> --to <name>` | Send message to specific member |
| `openhook channel send <channel> <message> --to all` | Broadcast to all members |

#### Channel Example

```bash
# Create a channel for agent coordination
openhook channel create deploy-team --description "Deployment coordination"

# Subscribe agents (each endpoint is an agent's webhook URL)
openhook channel subscribe ch_xxx --endpoint ep_deployer --name deployer
openhook channel subscribe ch_xxx --endpoint ep_monitor --name monitor
openhook channel subscribe ch_xxx --endpoint ep_notifier --name notifier

# Send targeted message
openhook channel send ch_xxx "deploy v2.1.0 to production" --to deployer

# Broadcast to all agents
openhook channel send ch_xxx "v2.1.0 is live" --to all --from deployer
```

## OpenClaw Integration

openhook has native OpenClaw support. Events are automatically forwarded to your AI agent:

```bash
# Start daemon with OpenClaw forwarding
openhook daemon start --openclaw
```

Or configure an HTTP endpoint in the dashboard with format "openclaw" to receive events.

## Links

- **Website**: https://openhook.dev
- **Dashboard**: https://openhook.dev/dashboard
- **Documentation**: https://docs.openhook.dev

## License

MIT

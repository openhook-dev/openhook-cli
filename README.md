# openhook CLI

Command-line interface for [openhook](https://openhook.dev) - the webhook relay service for AI agents.

Receive real-time webhook events from GitHub, Stripe, and Linear directly to your AI agent.

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
# 1. Authenticate (get API key from https://openhook.dev/dashboard)
openhook auth login --key oh_live_xxxxxxxx

# 2. Subscribe to events
openhook subscribe github --repo owner/repo --events push,pull_request

# 3. Start listening (daemon mode)
openhook daemon start --openclaw
```

## Commands

| Command | Description |
|---------|-------------|
| `openhook auth login --key <KEY>` | Authenticate with API key |
| `openhook auth status` | Check authentication status |
| `openhook subscribe github --repo owner/repo --events <events>` | Subscribe to GitHub events |
| `openhook subscribe stripe --events <events>` | Subscribe to Stripe events |
| `openhook subscribe linear --events <events>` | Subscribe to Linear events |
| `openhook list` | List active subscriptions |
| `openhook unsubscribe <ID>` | Remove a subscription |
| `openhook daemon start --openclaw` | Start background daemon |
| `openhook daemon status` | Check daemon status |
| `openhook daemon logs -f` | Tail daemon logs |
| `openhook daemon stop` | Stop daemon |

## OpenClaw Integration

openhook has native OpenClaw support. Events are automatically forwarded to your AI agent:

```bash
# Configure OpenClaw
npx openclaw config set hooks.enabled true
npx openclaw config set hooks.token "your-secret-token"

# Start daemon with OpenClaw forwarding
export OPENCLAW_HOOKS_TOKEN="your-secret-token"
openhook daemon start --openclaw
```

## Links

- **Website**: https://openhook.dev
- **Dashboard**: https://openhook.dev/dashboard
- **Documentation**: https://docs.openhook.dev
- **Skills**: `npx skills add openhook-dev/openhook-skills@openhook-webhook-listener`

## License

MIT

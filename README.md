# openhook CLI

Command-line interface for [openhook](https://openhook.dev), the webhook relay for AI agents.

Use the CLI to:
- authenticate with your OpenHook API key
- create subscriptions for GitHub, Stripe, and Linear events
- inspect event history
- open a real-time tunnel for incoming events
- forward events directly into OpenClaw
- run a background daemon for always-on event delivery

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap openhook-dev/openhook
brew install openhook
```

### Shell script

```bash
curl -fsSL https://openhook.dev/install.sh | sh
```

### Build from source

```bash
go install github.com/openhook-dev/openhook-cli/cmd/openhook@latest
```

## Quick start

### 1) Authenticate

Get an API key from <https://openhook.dev/dashboard> and log in:

```bash
openhook auth login --key oh_live_xxxxxxxx
```

Check your auth state anytime:

```bash
openhook auth status
openhook whoami
```

### 2) Subscribe to events

#### GitHub

```bash
openhook subscribe github \
  --repo openhook-dev/openhook \
  --events push,pull_request,issues
```

#### Stripe

```bash
openhook subscribe stripe \
  --events payment_intent.failed,invoice.paid
```

#### Linear

```bash
openhook subscribe linear \
  --team ENG \
  --events issue.created,issue.updated
```

List active subscriptions:

```bash
openhook list
openhook list --platform github
openhook list --json
```

### 3) Receive events in real time

Start a live tunnel:

```bash
openhook tunnel start
```

Or use the convenience alias:

```bash
openhook listen
```

Helpful flags:

```bash
openhook tunnel start --json
openhook tunnel start --timeout 10m
openhook tunnel start --max-events 25
```

### 4) Run in the background

```bash
openhook daemon start
openhook daemon status
openhook daemon logs -f
openhook daemon stop
```

## OpenClaw integration

`openhook` can forward webhook events directly into OpenClaw through `/hooks/agent`.

### Tunnel mode → OpenClaw

For local development or a workstation listener:

```bash
export OPENCLAW_HOOKS_TOKEN="your-secret-token"
openhook tunnel start --openclaw
```

Or explicitly set the gateway URL:

```bash
openhook tunnel start \
  --openclaw \
  --openclaw-url http://127.0.0.1:18789 \
  --openclaw-token "$OPENCLAW_HOOKS_TOKEN"
```

### Daemon mode → OpenClaw

For an always-on local bridge:

```bash
export OPENCLAW_HOOKS_TOKEN="your-secret-token"
openhook daemon start --openclaw
```

When OpenClaw forwarding is enabled, each event is wrapped into a message and sent to your local gateway so your agent can react to GitHub pushes, Stripe billing events, Linear issue updates, and more.

## Event history

Inspect recent events received by OpenHook:

```bash
openhook events list
openhook events list --source github --limit 10
openhook events list --status delivered
```

Get a single event by ID:

```bash
openhook events get evt_123
```

## Unsubscribe

Remove one subscription:

```bash
openhook unsubscribe sub_123
```

Remove all subscriptions for a platform:

```bash
openhook unsubscribe --all --platform github
openhook unsubscribe --all --platform github --yes
```

## Configuration

The CLI stores credentials under:

```text
~/.openhook/config.json
```

Daemon state/log files are also stored in `~/.openhook/`.

## Server override

If you need to point the CLI at a non-default server, many commands support:

```bash
--server https://your-openhook-server.example.com
```

Examples:

```bash
openhook auth status --server https://your-openhook-server.example.com
openhook list --server https://your-openhook-server.example.com
openhook tunnel start --server https://your-openhook-server.example.com
```

## Command reference

### Authentication
- `openhook auth login --key <KEY>`
- `openhook auth status`
- `openhook auth logout`
- `openhook whoami`

### Subscriptions
- `openhook subscribe github --repo owner/repo --events <events>`
- `openhook subscribe stripe --events <events>`
- `openhook subscribe linear --team <team> --events <events>`
- `openhook list`
- `openhook unsubscribe <ID>`
- `openhook unsubscribe --all --platform <platform>`

### Event delivery
- `openhook tunnel start`
- `openhook tunnel status`
- `openhook listen`
- `openhook daemon start`
- `openhook daemon status`
- `openhook daemon logs -f`
- `openhook daemon stop`

### Event history
- `openhook events list`
- `openhook events get <event-id>`

## Links

- Website: <https://openhook.dev>
- Dashboard: <https://openhook.dev/dashboard>
- Documentation: <https://docs.openhook.dev>
- OpenClaw skill: `npx skills add openhook-dev/openhook-skills@openhook-webhook-listener`

## License

MIT

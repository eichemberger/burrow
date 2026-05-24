# burrow documentation

**burrow** (`bin/burrow`) is an interactive CLI that sets up AWS SSM Session Manager port forwarding to private endpoints (RDS, ElastiCache, or any host) through an SSM-managed EC2 bastion.

## Guides

| Document | Description |
|----------|-------------|
| [Architecture](architecture.md) | How the app is structured, data flow, and key design decisions |
| [Services](services.md) | Built-in AWS service providers (RDS, ElastiCache, manual) |
| [Configuration](configuration.md) | Local config files, schemas, and recovery flows |
| [TUI & wizard](tui.md) | Interactive UI steps, startup gates, and error handling |

## Quick reference

```
~/.burrow/
├── config.yaml    # One-time EC2 bastion tag filters (TUI only)
└── targets.yaml   # Saved connection aliases
```

**Runtime split:** burrow uses the AWS SDK for discovery and validation, then delegates the actual port-forward session to the AWS CLI (`aws ssm start-session`) with the `AWS-StartPortForwardingSessionToRemoteHost` document.

**Entry points:**

- `./bin/burrow` — interactive TUI (default)
- `./bin/burrow --target <alias>` — connect to a saved target without TUI
- `./bin/burrow --list-targets` / `--show-target` / `--delete-target` — manage saved targets from the shell

See the [root README](../README.md) for install, flags, and keyboard shortcuts.

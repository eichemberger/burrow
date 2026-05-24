# burrow

Interactive TUI for AWS SSM Session Manager port forwarding to RDS, ElastiCache, OpenSearch, or any host/IP via an SSM-managed EC2 bastion.

Uses [`AWS-StartPortForwardingSessionToRemoteHost`](https://aws.amazon.com/blogs/aws/new-port-forwarding-using-aws-system-manager-sessions-manager/) so you can reach private endpoints without SSH or a dedicated jump host config on the target.

## Prerequisites

- **Go 1.25+** _or_ **Docker** (to build from source — see [Build with Docker](#build-with-docker-no-go-required) if you'd rather not install Go)
- **[AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)** on your `PATH`
- **[Session Manager plugin](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html)** installed
- AWS credentials (profile or environment variables) with permission to:
  - List/describe RDS, ElastiCache, OpenSearch, EC2, and SSM managed instances
  - Start SSM sessions on the chosen bastion instance
- At least one **SSM-managed EC2 instance** in the same VPC (or with network reachability) as the target

## Install

With **Go 1.25+** on your `PATH`:

```bash
go install github.com/eichemberger/burrow@latest
```

The binary is installed to `$(go env GOPATH)/bin/burrow`. Ensure that directory is on your `PATH`.

### Build from source

```bash
git clone https://github.com/eichemberger/burrow.git
cd burrow
make build
```

Binary: `bin/burrow`

### Build with Docker (no Go required)

If you don't have Go installed (or want to avoid pinning a specific version),
you can produce the `burrow` binary entirely inside a container. Only Docker
with BuildKit (default in modern Docker) is required on the host.

```bash
git clone https://github.com/eichemberger/burrow.git
cd burrow
make docker-build
```

…or, without `make`, pick the values that match your host (Go uses `darwin`,
`linux`, or `windows` for OS, and `amd64` or `arm64` for arch):

```bash
# Apple Silicon Mac
docker build \
  --build-arg TARGETOS=darwin \
  --build-arg TARGETARCH=arm64 \
  --output type=local,dest=./bin \
  --target export .

# Intel Mac
docker build \
  --build-arg TARGETOS=darwin --build-arg TARGETARCH=amd64 \
  --output type=local,dest=./bin --target export .

# Linux/amd64
docker build \
  --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 \
  --output type=local,dest=./bin --target export .
```

The freshly built binary is written to `./bin/burrow` on the host. The
`Makefile` target auto-detects your host OS/arch, so `make docker-build`
"just works" on macOS, Linux, and Apple Silicon.

To pin a specific Go toolchain version, pass `--build-arg GO_VERSION=1.25.5`.

> Note: `burrow` shells out to the AWS CLI and the Session Manager plugin, so
> the binary is meant to run **directly on your machine**, not inside the
> container. The Docker image is only used as a build environment.

### AWS SSO and MFA

burrow loads AWS credentials through the same profiles and environment variables as the AWS CLI and SDK. **SSO profiles** need a valid session before burrow can call AWS APIs — if yours has expired, run `aws sso login --profile <name>` (or your usual SSO login flow) and try again. **MFA profiles** may prompt for a one-time code in the terminal the first time credentials are loaded in a session; you have up to five minutes to enter it. If authentication times out, burrow prints a hint to retry with more time for MFA entry.

## Documentation

For architecture, service providers, configuration files, and TUI internals, see **[docs/](docs/README.md)**.

## Usage

```bash
./bin/burrow
```

### Saved connections

Connections are stored in `~/.burrow/targets.yaml`. EC2 bastion selection criteria are stored separately in `~/.burrow/config.yaml`. From the home screen:

- **Connect to a new server** — run the full setup wizard and optionally save
- **Connect to a saved connection** — pick a saved config and connect immediately
- **Manage connections** — add, edit, or delete saved configurations

### First-run setup

The first time you launch the interactive TUI (or if `config.yaml` is missing or invalid), burrow runs a one-time setup wizard. It asks for EC2 tag filters that identify which SSM-managed instances appear as bastion candidates in the connection wizard.

- Instances must match **all** configured tag filters (`key=value`, AND logic).
- You can add multiple tag filters during setup.
- Setup is saved to `~/.burrow/config.yaml` and is not asked again unless that file is removed or corrupted.
- CLI fast paths such as `--target` and `--list-targets` do not require `config.yaml`.

Example `config.yaml`:

```yaml
version: 1
selectors:
  ec2:
    tag_filters:
      - key: Role
        value: bastion
      - key: Environment
        value: production
```

To re-run setup, delete or fix `~/.burrow/config.yaml` and launch the TUI again.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--aws-dir` | `~/.aws` | Directory containing AWS `config` and `credentials` |
| `--burrow-dir` | `~/.burrow` | Directory for saved targets (`targets.yaml`) and setup config (`config.yaml`) |
| `--profile` | | AWS profile (skips profile picker) |
| `--region` | | AWS region (skips region picker) |
| `--target` | | Saved target alias — connect without opening the TUI |
| `--local-port` | | Override local bind port for `--target` (1–65535) |
| `--print` | `false` | Print the `aws ssm start-session` command instead of running it (requires `--target`) |
| `--background` | `false` | Detach after the tunnel is up and return to the shell (requires `--target`; POSIX only) |
| `--list-targets` | `false` | List saved targets and exit |
| `--show-target` | | Show one saved target and exit |
| `--delete-target` | | Delete a saved target and exit |
| `--manage` | `false` | Open the connection manager directly |
| `--debug` | `false` | Append debug output to `burrow-debug.log` |
| `--version` | | Print version and exit (set at build time via `-ldflags`) |

Examples:

```bash
# Interactive — home screen with saved targets and wizard
./bin/burrow

# Connect to a saved target
./bin/burrow --target my-db

# Override local port when the saved default is already in use
./bin/burrow --target my-db --local-port 15432

# Print the equivalent aws command (dry run)
./bin/burrow --target my-db --print

# Start a saved target in the background (POSIX only)
./bin/burrow --target my-db --background

# List / stop active background sessions
./bin/burrow status
./bin/burrow stop my-db
./bin/burrow stop --all

# List / inspect / delete saved targets
./bin/burrow --list-targets
./bin/burrow --show-target my-db
./bin/burrow --delete-target my-db

# Open connection manager (add / edit / delete)
./bin/burrow --manage

# Skip profile/region prompts
./bin/burrow --profile my-sso-profile --region us-east-1

# Custom AWS config directory
./bin/burrow --aws-dir /path/to/aws-config
```

### Background sessions

On macOS and Linux, `--background` runs preflight in the foreground (including MFA if needed), then starts the SSM session detached and exits once `localhost:<local-port>` is listening. Session metadata and logs live under `~/.burrow/sessions/`.

```bash
./bin/burrow --target my-db --background
./bin/burrow status
./bin/burrow status --json
./bin/burrow stop my-db
./bin/burrow stop 20260524T164300Z-abcd   # by session id
./bin/burrow stop --all
```

`burrow status` garbage-collects dead sessions automatically. A session is **unhealthy** when its process is still running but the local port is no longer accepting connections (for example, after credentials expire).

`--background` is not supported on Windows.

### Flow

**Home screen**

1. **Connect to a new server** — full setup wizard (optionally save when done)
2. **Connect to a saved connection** — pick a saved config and connect
3. **Manage connections** — add, edit, or delete saved configurations

**New connection wizard**

1. **Credentials** — use environment variables or pick a profile from `~/.aws`
2. **Region** — AWS region for API calls and the SSM session
3. **Service** — RDS, ElastiCache, OpenSearch, or manual host/IP
4. **Resource & endpoint** — e.g. Aurora writer/reader/custom endpoint, ElastiCache primary/reader/node
5. **Bastion** — pick an SSM-managed EC2 instance to tunnel through
6. **Local port** — port on `localhost` to bind
7. **Save (optional)** — give the connection an alias
8. **Session** — hands off to `aws ssm start-session` until you press `Ctrl+C`

**Manage connections**

- **Add via wizard** — same flow as Connect to a new server
- **Enter** on a target — Connect / Edit / Delete
- **e** — edit inline
- **d** — delete with confirmation

### Keys

- `↑` / `↓` — navigate
- `/` — search (RDS/ElastiCache/OpenSearch resources, endpoints, and bastion instances)
- `Enter` — select
- `Esc` — cancel search
- `b` / `Esc` (when not searching) — back
- `q` / `Ctrl+C` — quit (before session starts)

## Adding a new AWS service

The app is built around a pluggable `Provider` interface. To add a service (e.g. OpenSearch):

1. Create `internal/services/opensearch/opensearch.go`:

```go
package opensearch

import (
    "context"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/eichemberger/burrow/internal/services"
)

type Provider struct{}

func init() {
    services.Register(&Provider{})
}

func (p *Provider) Name() string { return "OpenSearch" }

func (p *Provider) ListResources(ctx context.Context, cfg aws.Config) ([]services.Resource, error) {
    // Call AWS APIs, then return one Resource per cluster/group with its endpoints.
    // VPCID and SecurityGroupIDs enable bastion reachability checks in the TUI.
    return []services.Resource{
        {
            Label: "my-domain (OpenSearch)",
            Endpoints: []services.Endpoint{
                {
                    Label: "VPC endpoint",
                    Target: services.Target{
                        Label:            "VPC endpoint",
                        Host:             "vpc-my-domain-abc123.us-east-1.es.amazonaws.com",
                        Port:             443,
                        VPCID:            "vpc-0123456789abcdef0",
                        SecurityGroupIDs: []string{"sg-0123456789abcdef0"},
                    },
                },
            },
        },
    }, nil
}
```

2. Blank-import the package in `main.go`:

```go
_ "github.com/eichemberger/burrow/internal/services/opensearch"
```

3. Run `go mod tidy` and rebuild.

No TUI changes are required — the new service appears automatically in the service menu.

### Data model

Each provider returns:

- `Resource` — something the user picks first (cluster, replication group, etc.)
- `Endpoint` — connection options within that resource (writer, reader, node, …)
- `Target` — `Host`, `Port`, optional `VPCID` (used to highlight matching bastions)

## Project layout

```
main.go
internal/
  awsconfig/       Profile discovery, SDK config loading
  bastion/         SSM + EC2 listing, SG reachability
  configstore/     ~/.burrow/config.yaml
  targetstore/     ~/.burrow/targets.yaml
  services/        Provider interface + registry
    rds/
    elasticache/
    opensearch/
  ssmexec/         CLI command builder, preflight, error classification
  runner/          Non-TUI --target connect
  session/         Background session registry, spawn, and status
  cli/             status and stop subcommands
  tui/             Root Bubble Tea app (step routing)
    steps/         One model per wizard screen
  ui/              Shared lipgloss styles and page chrome
  netutil/         IP / CIDR helpers
```

## Development

```bash
make fmt    # go fmt
make vet    # go vet
make test   # go test ./...
make tidy   # go mod tidy
make clean  # remove bin/
make run    # build and run
```

## License

MIT

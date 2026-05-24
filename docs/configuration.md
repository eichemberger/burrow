# Configuration

burrow stores all local state under **`~/.burrow`** (override with `--burrow-dir`).

```
~/.burrow/
├── config.yaml     # EC2 bastion selection criteria (one-time setup)
└── targets.yaml    # Saved connection profiles
```

Both files use YAML with an explicit `version` field for forward compatibility.

---

## config.yaml — bastion selection

**Managed by:** one-time TUI setup wizard  
**Required for:** TUI bastion discovery only  
**Not required for:** `--target`, `--list-targets`, and other CLI fast paths

### Schema (v1)

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

| Field | Description |
|-------|-------------|
| `selectors.ec2.tag_filters` | List of EC2 tag key/value pairs. Instances must match **all** filters (AND logic). |
| `selectors.<type>` | Reserved for future resource types (e.g. additional selectors beyond EC2). |

### When setup runs

The setup wizard appears when:

- `config.yaml` does not exist (first TUI launch)
- `config.yaml` is invalid YAML or fails validation

To re-run setup, delete or fix the file and launch the TUI again.

### How filters are applied

During bastion discovery (`internal/bastion/bastion.go`):

1. All SSM-managed instance IDs are collected.
2. EC2 `DescribeInstances` returns tags for each instance.
3. Instances not matching every configured tag are excluded.
4. Remaining instances proceed to reachability filtering.

---

## targets.yaml — saved connections

**Managed by:** TUI wizard (optional save step), connection manager, CLI `--delete-target`  
**Required for:** saved connections and `--target`

### Schema (v1)

```yaml
version: 1
targets:
  my-db:
    aws_profile: dev-sso
    region: us-east-1
    bastion_id: i-0abc123def456
    host: my-db.cluster-xyz.us-east-1.rds.amazonaws.com
    remote_port: 5432
    local_port: 15432
    description: Aurora writer
  prod-redis:
    use_env: true
    region: us-west-2
    bastion_id: i-0def456abc789
    host: my-redis.xxxxx.cache.amazonaws.com
    remote_port: 6379
    local_port: 16379
```

| Field | Required | Description |
|-------|----------|-------------|
| `aws_profile` | If `use_env` is false | AWS shared credentials profile name |
| `use_env` | No | When true, use environment/instance credentials instead of a profile |
| `region` | Yes | AWS region for API calls and SSM session |
| `bastion_id` | Yes | EC2 instance ID of the SSM bastion |
| `host` | Yes | Remote hostname or IP to forward to |
| `remote_port` | Yes | Port on the remote host (1–65535) |
| `local_port` | Yes | Port to bind on localhost (1–65535) |
| `description` | No | Free-text note shown in lists |

Alias keys must match `^[a-zA-Z0-9][a-zA-Z0-9_-]*$` (max 64 characters).

---

## Recovery flows

### Corrupt targets.yaml

If `targets.yaml` cannot be read or parsed, the TUI shows a **recovery screen** instead of exiting:

1. Displays the file path and parse error.
2. Recommends opening the file in an editor to fix YAML manually.
3. Offers **reset** (`r`) — replaces the file with an empty config after a confirmation that **all saved connections will be permanently deleted**.

CLI commands that read targets (`--target`, `--list-targets`) will still fail with an error until the file is fixed or reset via the TUI.

### Corrupt config.yaml

Invalid or missing `config.yaml` triggers the **EC2 setup wizard** again (same as first run). This does not affect saved connections in `targets.yaml`.

---

## Environment & paths

| Flag / env | Default | Effect |
|------------|---------|--------|
| `--burrow-dir` | `~/.burrow` | Directory for both YAML files |
| `--aws-dir` | `~/.aws` | AWS shared config/credentials directory |
| `AWS_CONFIG_FILE` | — | If set, `aws-dir` defaults to its parent directory |
| `AWS_PROFILE` | — | Can be selected in TUI or stored per target |
| Standard AWS credential env vars | — | Used when `use_env: true` or env-auth is chosen |

---

## What gets persisted vs. discovered

| Data | Stored on disk? | Source at runtime |
|------|-----------------|-------------------|
| Bastion tag filters | Yes (`config.yaml`) | Setup wizard |
| Connection alias, host, ports, bastion ID | Yes (`targets.yaml`) | Wizard or manager |
| RDS / ElastiCache resource lists | No | AWS APIs (live) |
| SSM bastion online status | No | AWS APIs (live, preflight before session) |
| Security group reachability | No | Computed during wizard |

Saved connections store a **snapshot** of bastion ID and endpoint. If the bastion is later deleted, terminated, or your credentials point at a different account, the session preflight fails with guidance to create a new connection. See [TUI & wizard](tui.md#error-handling).

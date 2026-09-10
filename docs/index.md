# GreenNode CLI

Universal Command Line Interface for GreenNode.

The GreenNode CLI (`grn`) is a unified tool to manage your GreenNode services from the command line. Written in Go, distributed as a single binary with zero dependencies.

## Quick Start

Install with one command (macOS / Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.sh | bash
```

See [Installation](installation.md) for Windows and build-from-source.

```bash
# Configure credentials
grn configure

# List your VKS clusters
grn vks list-clusters

# Get cluster details
grn vks get-cluster --cluster-id <id>
```

## Features

- **Single Binary** — Zero dependencies, download and run
- **VKS Management** — Full cluster and node group lifecycle (create, get, update, delete)
- **[vBackup Gateway](commands/vbackup/index.md)** — HCM-3 policies, backup servers, restore points, and history
- **Multiple Output Formats** — JSON, table, and text with JMESPath query filtering
- **Auto-pagination** — VKS list commands fetch all pages by default
- **Dry-run** — Validate parameters before create/update/delete
- **Delete Confirmation** — Preview and confirm before destructive operations
- **Waiter Commands** — Wait for async operations to complete
- **Profile Support** — Multiple credential profiles for different environments
- **Read-only Retry** — Shared transport retries transient reads, never mutations; AgentBase has separate transport
- **Security** — Credentials masked in output, input validation, SSL by default
- **Cross-platform** — Linux, macOS, Windows (amd64, arm64)

## Additional Services

| Command | Scope |
| --- | --- |
| [iam](commands/iam/index.md) | Accounts, credentials, groups, policies, and bindings |
| [vbackup](commands/vbackup/index.md) | Backup policies, servers, and restore points |
| [vcr](commands/vcr/index.md) | Container repositories, images, and users |
| [vdb](commands/vdb/index.md) | Relational, MemoryStore, Kafka, and PostgreSQL |
| [vlb](commands/vlb/index.md) | Load balancers, listeners, pools, and certificates |
| [vmonitor](commands/vmonitor/index.md) | Dashboards, alarms, infrastructure, and metrics |
| [vmonitor-log](commands/vmonitorlog/index.md) | Log search, pipelines, archives, and mappings |
| [vstorage](commands/vstorage/index.md) | Regional containers, buckets, and objects |
| [vstorage-gateway](commands/vstoragegateway/index.md) | Storage region, project, and container discovery |
| [saas-ai](commands/saasai/index.md) | Speech transcription and synthesis |

These services have fixture-backed local tests. See each reference for endpoint differences, safety limits, and unverified live behavior.

## Adding New Services

See [Architecture & Adding a Service](development/architecture.md) for the structure and integration steps.

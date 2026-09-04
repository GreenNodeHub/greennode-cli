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
- **Multiple Output Formats** — JSON, table, and text with JMESPath query filtering
- **Auto-pagination** — List commands fetch all pages by default
- **Dry-run** — Validate parameters before create/update/delete
- **Delete Confirmation** — Preview and confirm before destructive operations
- **Waiter Commands** — Wait for async operations to complete
- **Profile Support** — Multiple credential profiles for different environments
- **Retry with Backoff** — Automatic retry for transient errors (5xx, timeouts)
- **Security** — Credentials masked in output, input validation, SSL by default
- **Cross-platform** — Linux, macOS, Windows (amd64, arm64)

## Adding New Services

Other product teams can add a CLI for their service in this repo without
conflicting with each other. See [Architecture & Adding a Service](development/architecture.md)
for the structure and step-by-step guide.

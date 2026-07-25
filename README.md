# GreenNode CLI

The GreenNode CLI (`grn`) is a unified tool to manage your GreenNode services from the command line.

- [Getting Started](#getting-started)
- [Getting Help](#getting-help)
- [More Resources](#more-resources)

## Getting Started

### Requirements

- No dependencies required — `grn` is a single binary

### Installation

Download the latest binary for your platform from [GitHub Releases](https://github.com/vngcloud/greennode-cli/releases):

**macOS / Linux:**

```bash
# Download (replace OS and ARCH as needed)
curl -L -o grn https://github.com/vngcloud/greennode-cli/releases/latest/download/grn-darwin-arm64
chmod +x grn
sudo mv grn /usr/local/bin/
```

**Or build from source:**

```bash
git clone https://github.com/vngcloud/greennode-cli.git
cd greennode-cli/go
go build -o grn .
sudo mv grn /usr/local/bin/
```

**Verify installation:**

```bash
grn --version
# grn-cli/0.1.0 Go/1.22.2 darwin/arm64
```

### Configuration

Before using the GreenNode CLI, you need to configure your credentials. There are three ways:

**Method 1: Environment variables**

```bash
export GRN_ACCESS_KEY_ID=your-client-id
export GRN_SECRET_ACCESS_KEY=your-client-secret
export GRN_DEFAULT_REGION=HCM-3
export GRN_DEFAULT_PROJECT_ID=pro-xxxxxxxx   # optional
```

**Method 2: Interactive setup (recommended)**

```bash
grn configure
```

```
GRN Client ID [None]: <your-client-id>
GRN Client Secret [None]: <your-client-secret>
Default region name [HCM-3]:
Default output format [json]:
Project ID (leave blank to auto-detect) [None]:
Fetching project_id from HCM-3...
Auto-detected project_id: pro-xxxxxxxx
```

**Method 3: Credentials file (manual)**

```ini
# ~/.greennode/credentials
[default]
client_id = your-client-id
client_secret = your-client-secret
```

```ini
# ~/.greennode/config
[default]
region = HCM-3
output = json
project_id = pro-xxxxxxxx
```

Credentials are obtained from the [GreenNode IAM Portal](https://hcm-3.console.vngcloud.vn/iam/) under Service Accounts.

Credential resolution order: environment variables take priority over the credentials file.

To use multiple profiles:

```bash
grn configure --profile staging
grn --profile staging vks list-clusters
```

For more configuration options, see the [Configuration Guide](https://vngcloud.github.io/greennode-cli/configuration/).

### Basic Commands

The GreenNode CLI uses a multi-part command structure:

```bash
grn <service> <command> [options and parameters]
```

For example, to list your VKS clusters:

```bash
grn vks list-clusters
```

To get help on any command:

```bash
grn help
grn vks
grn vks create-cluster --help
```

To check the version:

```bash
grn --version
```

### Available VKS Commands

**Cluster**

- `list-clusters` — List all VKS clusters
- `get-cluster` — Get cluster details
- `create-cluster` — Create a new VKS cluster
- `update-cluster` — Update a VKS cluster
- `delete-cluster` — Delete a VKS cluster

**Node Group**

- `list-nodegroups` — List node groups for a cluster
- `get-nodegroup` — Get node group details
- `create-nodegroup` — Create a new node group
- `update-nodegroup` — Update a node group
- `update-nodegroup-metadata` — Update labels, tags, and taints of a node group
- `upgrade-nodegroup-version` — Upgrade the Kubernetes version of a node group
- `list-nodes` — List nodes in a node group
- `delete-nodegroup` — Delete a node group

**Versions**

- `list-cluster-versions` — List available Kubernetes versions

**Auto-Upgrade**

- `set-auto-upgrade-config` — Configure auto-upgrade schedule for a cluster
- `delete-auto-upgrade-config` — Delete auto-upgrade config for a cluster

**Auto-Healing**

- `config-auto-healing` — Configure auto-healing for a cluster

**Events**

- `get-cluster-events` — Get the list of events for a cluster

**Kubeconfig**

- `generate-kubeconfig` — Request generation of a cluster kubeconfig
- `update-kubeconfig` — Fetch and merge the cluster kubeconfig into your kubeconfig file

**Quota**

- `get-quota` — Get VKS quota limits and current usage

**Waiter**

- `wait cluster-active` — Wait until a cluster reaches ACTIVE status
- `wait cluster-deleted` — Wait until a cluster is fully deleted
- `wait nodegroup-active` — Wait until a node group reaches ACTIVE status
- `wait nodegroup-deleted` — Wait until a node group is fully deleted

## Getting Help

The best way to interact with our team is through GitHub:

- [Open an issue](https://github.com/vngcloud/greennode-cli/issues/new/choose) — Bug reports and feature requests
- Search [existing issues](https://github.com/vngcloud/greennode-cli/issues) before opening a new one

## More Resources

- [Documentation](https://vngcloud.github.io/greennode-cli/)
- [Changelog](CHANGELOG.md)
- [Contributing Guide](CONTRIBUTING.md)
- [GreenNode Console](https://hcm-3.console.vngcloud.vn/)

## License

Apache License 2.0 — see [LICENSE](LICENSE).

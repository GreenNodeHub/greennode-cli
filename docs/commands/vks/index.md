# VKS Commands

VKS (GreenNode Kubernetes Service) commands for managing Kubernetes clusters and node groups.

```bash
grn vks <command> [options]
```

## Available commands

### Cluster

| Command | Description |
|---------|-------------|
| [list-clusters](list-clusters.md) | List all VKS clusters |
| [get-cluster](get-cluster.md) | Get cluster details |
| [create-cluster](create-cluster.md) | Create a new VKS cluster |
| [update-cluster](update-cluster.md) | Update a VKS cluster |
| [delete-cluster](delete-cluster.md) | Delete a VKS cluster |
| [get-upgrade-insights](get-upgrade-insights.md) | Get cluster upgrade insights |
| [stop-poc](stop-poc.md) | End cluster proof-of-concept mode |

### Fleet Management

| Command | Description |
|---------|-------------|
| [register-fleet](register-fleet.md) | Register a cluster with fleet management |
| [unregister-fleet](unregister-fleet.md) | Remove a cluster's fleet association |

### Node Group

| Command | Description |
|---------|-------------|
| [list-nodegroups](list-nodegroups.md) | List node groups for a cluster |
| [get-nodegroup](get-nodegroup.md) | Get node group details |
| [create-nodegroup](create-nodegroup.md) | Create a new node group |
| [update-nodegroup](update-nodegroup.md) | Update a node group |
| [update-nodegroup-metadata](update-nodegroup-metadata.md) | Update labels, tags, and taints of a node group |
| [upgrade-nodegroup-version](upgrade-nodegroup-version.md) | Upgrade the Kubernetes version of a node group |
| [list-nodes](list-nodes.md) | List nodes in a node group |
| [delete-nodegroup](delete-nodegroup.md) | Delete a node group |
| [get-nodegroup-events](get-nodegroup-events.md) | Get node-group events |
| [list-nodegroup-images](list-nodegroup-images.md) | List node-group images |

### Versions

| Command | Description |
|---------|-------------|
| [list-cluster-versions](list-cluster-versions.md) | List available Kubernetes versions |

### Auto-Upgrade

| Command | Description |
|---------|-------------|
| [config-auto-upgrade](config-auto-upgrade.md) | Configure auto-upgrade schedule for a cluster (alias: set-auto-upgrade-config) |
| [delete-auto-upgrade-config](delete-auto-upgrade-config.md) | Delete auto-upgrade config for a cluster |

### Auto-Healing

| Command | Description |
|---------|-------------|
| [config-auto-healing](config-auto-healing.md) | Configure auto-healing for a cluster |

### Events

| Command | Description |
|---------|-------------|
| [get-cluster-events](get-cluster-events.md) | Get the list of events for a cluster |

### Kubeconfig

| Command | Description |
|---------|-------------|
| [generate-kubeconfig](generate-kubeconfig.md) | Request generation of a cluster kubeconfig |
| [update-kubeconfig](update-kubeconfig.md) | Fetch and merge the cluster kubeconfig into your kubeconfig file |
| [acknowledge-kubeconfig-warning](acknowledge-kubeconfig-warning.md) | Acknowledge a kubeconfig renewal warning |

### Quota

| Command | Description |
|---------|-------------|
| [get-quota](get-quota.md) | Get VKS quota limits and current usage |

### Workspace

| Command | Description |
|---------|-------------|
| [get-workspace](get-workspace.md) | Get the current VKS workspace |
| [create-workspace](create-workspace.md) | Create a VKS workspace |
| [reset-workspace-service-account](reset-workspace-service-account.md) | Reset the workspace service account |

### Waiter

| Command | Description |
|---------|-------------|
| [wait](wait.md) | Wait until a cluster or node group reaches a desired state |

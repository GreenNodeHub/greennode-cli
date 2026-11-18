# VKS Commands

VKS (GreenNode Kubernetes Service) commands for managing Kubernetes clusters, node groups and cluster add-ons.

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

### Versions

| Command | Description |
|---------|-------------|
| [list-cluster-versions](list-cluster-versions.md) | List available Kubernetes versions |

### Add-on

| Command | Description |
|---------|-------------|
| [list-addon-versions](list-addon-versions.md) | List the add-ons of the catalog and their versions |
| [get-addon-configuration-schema](get-addon-configuration-schema.md) | Get the configuration schema of an add-on version |
| [list-addons](list-addons.md) | List the add-ons installed on a cluster |
| [get-addon](get-addon.md) | Get details of an add-on installed on a cluster |
| [create-addon](create-addon.md) | Install an add-on on a cluster |
| [update-addon](update-addon.md) | Update the version or configuration of an add-on |
| [delete-addon](delete-addon.md) | Delete an add-on from a cluster |

### Update

| Command | Description |
|---------|-------------|
| [list-updates](list-updates.md) | List the update IDs of a cluster, a node group or an add-on |
| [get-update](get-update.md) | Get the status of an update |

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

### Quota

| Command | Description |
|---------|-------------|
| [get-quota](get-quota.md) | Get VKS quota limits and current usage |

### Waiter

| Command | Description |
|---------|-------------|
| [wait](wait.md) | Wait until a cluster, node group, add-on or update reaches a desired state |

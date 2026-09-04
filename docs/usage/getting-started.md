# Getting Started

## Help

```bash
grn                              # Show available commands
grn help                         # Same
grn vks                          # Show available VKS commands
grn vks create-cluster --help    # Show command help with all flags
```

## Command structure

```
grn [global-options] <service> <command> [command-options]
```

Example:

```bash
grn --profile staging --output table vks list-clusters --page-size 20
```

## Basic commands

```bash
# List clusters
grn vks list-clusters

# Get cluster details
grn vks get-cluster --cluster-id k8s-xxxxx

# Create a cluster (control plane only). Pass one ID to --subnet-ids for a
# single-subnet cluster, or several comma-separated for a multi-subnet one.
grn vks create-cluster \
  --name my-cluster \
  --k8s-version v1.30.10-vks.1746550800 \
  --network-type CILIUM_OVERLAY \
  --vpc-id net-xxxxx \
  --subnet-ids sub-xxxxx \
  --cidr 192.168.0.0/16

# Add worker nodes — node groups are a separate command
grn vks create-nodegroup \
  --cluster-id k8s-xxxxx \
  --name default \
  --flavor-id flav-xxxxx \
  --os ubuntu \
  --disk-type vtype-xxxxx \
  --ssh-key-id ssh-xxxxx

# Delete a cluster (prompts for confirmation)
grn vks delete-cluster --cluster-id k8s-xxxxx

# Wait for cluster to be ready
grn vks wait cluster-active --cluster-id k8s-xxxxx
```

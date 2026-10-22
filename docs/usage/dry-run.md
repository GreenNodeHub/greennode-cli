# Dry-run & Confirmation

## Dry-run

Commands that expose `--dry-run` preview their action without performing the mutation:

```bash
# Validate create parameters without calling API
grn vks create-cluster --dry-run --name my-cluster --k8s-version v1.30 ...
grn vks create-nodegroup --dry-run --cluster-id k8s-xxxxx --name workers ...

# Preview update parameters
grn vks update-cluster --dry-run --cluster-id k8s-xxxxx --k8s-version v1.31 --whitelist-node-cidrs 0.0.0.0/0
grn vks update-nodegroup --dry-run --cluster-id k8s-xxxxx --nodegroup-id ng-xxxxx --num-nodes 3

# Preview what will be deleted
grn vks delete-cluster --dry-run --cluster-id k8s-xxxxx
grn vks delete-nodegroup --dry-run --cluster-id k8s-xxxxx --nodegroup-id ng-xxxxx
```

### Create dry-run

Validates parameters offline:

- Cluster/nodegroup name format
- Disk size range (20-5000 GiB)
- Number of nodes range (0-10)
- CIDR requirement for TIGERA/CILIUM_OVERLAY networks

### Delete dry-run

Validates identifiers and displays the request target without loading credentials or fetching the resource:

```
=== DRY RUN ===
Would delete VKS cluster k8s-xxxxx:
  method: DELETE
  path: /v1/clusters/k8s-xxxxx

Run without --dry-run to delete.
```

## Delete confirmation

Delete commands show a preview and prompt for confirmation:

```bash
grn vks delete-cluster --cluster-id k8s-xxxxx
# The following resources will be deleted:
# ...
# Are you sure you want to delete this cluster? (yes/no): yes
```

### Skip confirmation

Use `--force` to skip the confirmation prompt (for scripting):

```bash
grn vks delete-cluster --cluster-id k8s-xxxxx --force
```

## Non-interactive automation

Add `--non-interactive` to prevent prompts. A destructive confirmation without `--force` returns a failure exit status and leaves stdin untouched. `configure` and browser `login` refuse this mode; use `configure set` for scripted configuration. AgentBase's `--interactive` does not override `--non-interactive`. Placement-group creation requires an explicit `--policy-id` in this mode.

Every dry-run is fully offline: it validates inputs and prints a redacted preview before configuration, client construction, network requests, confirmation, or output-file writes. Check each command's help for supported safety flags.

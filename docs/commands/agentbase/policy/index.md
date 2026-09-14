# policy

Manage Cedar authorization policy groups, policies, and decisions (the policy service).

A "policy group" is a container of policies owned by a user (max 20/user); each
group holds "policies" — individual permit/forbid rules (max 10/group) compiled
to Cedar at write time. A gateway binds a group via its `policyGroupId`, and its
enforcement asks the policy service for an allow/deny decision per inbound request.
Policy resources are **synchronous** (no `WAITING_*` FSM, so there is no `wait`).
There are two nested resources: **policy groups** and **policies within a group**.

```bash
grn agentbase policy <command> [options]
```

## Available commands

### Policy Group

| Command | Description |
|---------|-------------|
| [group create](group-create.md) | Create a policy group |
| [group generate](group-generate.md) | Print a policy-group create template (YAML or JSON) |
| [group list](group-list.md) | List policy groups |
| [group get](group-get.md) | Show a policy group |
| [group update](group-update.md) | Update a policy group |
| [group delete](group-delete.md) | Delete a policy group (cascades to its policies) |

### Group Policy

| Command | Description |
|---------|-------------|
| [group policy create](group-policy-create.md) | Create a policy within a group |
| [group policy generate](group-policy-generate.md) | Print a policy create template (YAML or JSON) |
| [group policy list](group-policy-list.md) | List policies within a group |
| [group policy get](group-policy-get.md) | Show a policy |
| [group policy update](group-policy-update.md) | Update a policy (merge-patch) |
| [group policy delete](group-policy-delete.md) | Delete a policy |

### Other

| Command | Description |
|---------|-------------|
| [condition-operators](condition-operators.md) | List accepted policy condition operators |
| [decide](decide.md) | Probe an authorization decision for a gateway target |

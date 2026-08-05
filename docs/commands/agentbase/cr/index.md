# cr

Manage the agentbase container registry (agent-core-container-registry).

```bash
grn agentbase cr <command> [options]
```

Each user gets an auto-provisioned repository and robot account on first access —
there is no `create` command and no finite-state machine: the resources appear
when first read. The robot account (username + secret) is what you use for
`docker login`. Delete operations identify their target by a query parameter
(`?imageName=`, `?digest=`) and return `204 No Content` on success.

The robot-account secret authorizes push/pull, so it is real — it is MASKED in
table output (last-4 shown) and revealed only with `-o json`.

## Available commands

### Repository

| Command | Description |
|---------|-------------|
| [repository get](repository-get.md) | Show the user's repository info |
| [repository image list](repository-image-list.md) | List images |
| [repository image delete](repository-image-delete.md) | Delete an image (all its artifacts/tags) |
| [repository artifact list](repository-artifact-list.md) | List artifacts within an image |
| [repository artifact delete](repository-artifact-delete.md) | Delete a single artifact by digest |

### Registry Credential

| Command | Description |
|---------|-------------|
| [registry-credential get](registry-credential-get.md) | Show the robot account (username + secret) |
| [registry-credential reset-secret](registry-credential-reset-secret.md) | Rotate the robot-account secret |

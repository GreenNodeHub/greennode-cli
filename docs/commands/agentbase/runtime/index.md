# runtime

Manage agent runtimes.

```bash
grn agentbase runtime <command> [options]
```

An agent runtime is a deployable container defined by its image, command, args,
environment, autoscaling, and flavor. Runtimes converge asynchronously:
`create`/`update` return as soon as the service accepts the spec (state
`CREATING`/`UPDATING`) and reach `ACTIVE` (or `ERROR` / `SERVICE_ACCOUNT_ERROR`).
Use [wait](wait.md) to block until terminal.

Runtimes are addressed by `id` (the `name` is immutable after creation). `update`
is a **full-spec replacement** — every field is required (the create spec minus
`name`), not a merge-patch. For anything beyond the simple path, generate a
template with [generate](generate.md), fill it in, and apply with `--file`.

## Available commands

| Command | Description |
|---------|-------------|
| [create](create.md) | Create a new agent runtime |
| [generate](generate.md) | Print a runtime create template (YAML or JSON) |
| [list](list.md) | List agent runtimes |
| [get](get.md) | Show an agent runtime |
| [update](update.md) | Update an agent runtime (full-spec replacement) |
| [delete](delete.md) | Delete an agent runtime |
| [wait](wait.md) | Wait for an agent runtime to reach a terminal state |

## Sub-resource groups

| Group | Commands |
|-------|----------|
| [endpoint](endpoint-list.md) | [`list`](endpoint-list.md) · [`create`](endpoint-create.md) · [`update`](endpoint-update.md) · [`delete`](endpoint-delete.md) · [`start`](endpoint-start.md) · [`stop`](endpoint-stop.md) · [`logs`](endpoint-logs.md) · [`metrics`](endpoint-metrics.md) · [`events`](endpoint-events.md) |
| runtime | [`logs`](logs.md) · [`reset-service-account`](reset-service-account.md) · [`versions`](versions.md) |
| [trace](trace-get.md) | [`get`](trace-get.md) · [`search`](trace-search.md) · [`tag-values`](trace-tag-values.md) |

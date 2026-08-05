# deploy destroy

Delete an agent's runtime and memory (and identity with `--purge`).

## Description

Tear down an agent by name (best-effort, reverse of apply):

1. **runtime** — delete by id, wait for `DELETED`.
2. **memory** — soft-delete by id (`ACTIVE` → `DELETED`).
3. **identity** — only with `--purge` (it may be referenced by other agents).

Missing sub-resources are reported and skipped; each step's outcome is shown. The runtime deletion is asynchronous, so `destroy` waits for it to reach `DELETED` (use `--timeout` to bound the wait).

Resources are looked up fresh by name on each invocation — there is no state file. The teardown is **best-effort**: a failure on one step is reported but does not stop later steps.

## Synopsis

```text
grn agentbase deploy destroy <name>
    [--purge]
    [--timeout <value>]
    [--interval <value>]
```

## Options

**`--purge`** (boolean)

Also delete the identity (default: leave it).

- Required: No
- Default: `false`

**`--timeout`** (duration)

Maximum time to wait for the runtime to delete.

- Required: No
- Default: `10m`

**`--interval`** (duration)

Poll interval while deleting.

- Required: No
- Default: `5s`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Delete an agent's runtime and memory, keeping the identity:

```bash
grn agentbase deploy destroy my-agent
```

Tear down everything, including the identity:

```bash
grn agentbase deploy destroy my-agent --purge
```

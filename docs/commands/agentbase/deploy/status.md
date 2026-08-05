# deploy status

Show the cross-service state of an agent.

## Description

Show the cross-service state of an agent, looked up by name across the identity, memory, and runtime clients. Each service is reported as `present`, `absent`, or `error`, with its id and (for the runtime) its async `Status` (e.g. `CREATING`, `ACTIVE`, `DELETED`).

Memory and runtime have no by-name lookup, so `status` pages their lists and filters client-side. The identity is fetched by name directly. Resources are looked up fresh on each invocation — there is no state file.

## Synopsis

```text
grn agentbase deploy status <name>
    [--timeout <value>]
```

## Options

**`--timeout`** (duration)

Per-service lookup timeout (reserved).

- Required: No
- Default: `10s`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Show the state of an agent:

```bash
grn agentbase deploy status my-agent
```

Output the rollup as JSON:

```bash
grn agentbase deploy status my-agent -o json
```

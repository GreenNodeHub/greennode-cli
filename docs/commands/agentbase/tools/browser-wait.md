# tools browser wait

Wait for a browser to become ACTIVE.

## Description

Poll a browser (GET `/v1/browsers/{id}`) until it reaches a terminal state.
Use after `create` — sessions can only attach to an `ACTIVE` browser.

Terminal states:

| Outcome | States |
|---------|--------|
| Success | `ACTIVE` |
| Failure | `FAILED`, `DELETED`, timeout |

Progress lines go to stderr. On failure/timeout the command exits non-zero.

## Synopsis

```text
grn agentbase tools browser wait <id>
    [--timeout <value>]
    [--interval <value>]
```

## Arguments

**`<id>`** (string) — browser id. Required.

## Options

**`--timeout`** (duration) — maximum time to wait. Default `10m`.

**`--interval`** (duration) — interval between polls. Default `5s`.

## Output

- `table` — a confirmation line on stdout
- `json` — `{"ID": "<id>"}`
- `id` — the browser's id

## Examples

```bash
grn agentbase tools browser create --name my-browser -o id
grn agentbase tools browser wait br-1a2b3c
grn agentbase tools browser wait br-1a2b3c --interval 10s --timeout 30m
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

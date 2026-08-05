# runtime wait

Wait for an agent runtime to reach a terminal state.

## Description

Poll an agent runtime until it reaches a terminal state: `ACTIVE` / `DELETED`
(success) or `ERROR` / `SERVICE_ACCOUNT_ERROR` (failure). Use after
[create](create.md), [update](update.md), or [delete](delete.md).

Progress lines are written to stderr. On failure or timeout the command returns a
non-zero exit status.

## Synopsis

```text
grn agentbase runtime wait <id>
    [--timeout <duration>]
    [--interval <duration>]
```

## Options

**`--timeout`** (duration)

Maximum time to wait before giving up.

- Required: No
- Default: `10m0s` (10 minutes)
- Possible values: any Go duration (e.g. `5m`, `30s`, `1h`).

**`--interval`** (duration)

Poll interval between get requests.

- Required: No
- Default: `5s` (5 seconds)
- Possible values: any Go duration (e.g. `5s`, `10s`, `1m`).

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Wait for a freshly created runtime to become `ACTIVE`:

```bash
grn agentbase runtime create --file rt.yaml
grn agentbase runtime wait <id>
```

Wait with a custom timeout and poll interval:

```bash
grn agentbase runtime wait agn-rt-abc123 --timeout 30m --interval 10s
```

Wait for deletion to complete (reaches `DELETED`):

```bash
grn agentbase runtime delete agn-rt-abc123
grn agentbase runtime wait agn-rt-abc123
```

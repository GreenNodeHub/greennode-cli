# gateway wait

Wait for a gateway to reach a terminal state.

## Description

Poll a gateway until it reaches a terminal state. Use after `create`, `update`,
or `delete`.

Terminal states:

| Outcome | States |
|---------|--------|
| Success | `ACTIVE`, `DELETED` |
| Failure | `CREATE_ERROR`, `UPDATE_ERROR`, `ERROR`, `ERROR_DELETING` |

On a failure state the command prints the gateway's last error (code, message,
stage) and exits non-zero. On timeout it exits non-zero with the last observed
state. All non-terminal states (`WAITING_*`, `CREATING`, `UPDATING`,
`DELETING`, `*_CLEANING`) keep polling.

## Synopsis

```text
grn agentbase gateway wait <name>
    [--timeout <value>]
    [--interval <value>]
```

## Arguments

**`<name>`** (string)

Name of the gateway to poll.

- Required: Yes (exactly one positional argument)

## Options

**`--timeout`** (duration)

Maximum time to wait before timing out.

- Required: No
- Default: `10m`
- Syntax: Go duration, e.g. `10m`, `30m`, `2h`

**`--interval`** (duration)

Interval between poll (GET) attempts.

- Required: No
- Default: `5s`
- Syntax: Go duration, e.g. `5s`, `10s`, `1m`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Wait for a freshly created gateway to become ACTIVE:

```bash
grn agentbase gateway create --file gw.yaml
grn agentbase gateway wait my-gateway
```

Wait for a deletion to finish (terminates on `DELETED`), polling every 10
seconds up to 30 minutes:

```bash
grn agentbase gateway delete my-gateway
grn agentbase gateway wait my-gateway --interval 10s --timeout 30m
```

# tools code-interpreter wait

Wait for a code interpreter to become ACTIVE.

## Description

Poll a code interpreter (GET `/v1/code-interpreters/{id}`) until it reaches a
terminal state. Use after `create` — sessions can only attach to an `ACTIVE`
code interpreter.

Terminal states:

| Outcome | States |
|---------|--------|
| Success | `ACTIVE` |
| Failure | `FAILED`, `DELETED`, timeout |

Progress lines go to stderr. On failure/timeout the command exits non-zero.

## Synopsis

```text
grn agentbase tools code-interpreter wait <id>
    [--timeout <value>]
    [--interval <value>]
```

## Arguments

**`<id>`** (string) — code interpreter id. Required.

## Options

**`--timeout`** (duration) — maximum time to wait. Default `10m`.

**`--interval`** (duration) — interval between polls. Default `5s`.

## Output

- `table` — a confirmation line on stdout
- `json` — `{"ID": "<id>"}`
- `id` — the code interpreter's id

## Examples

```bash
grn agentbase tools code-interpreter create --name my-runner -o id
grn agentbase tools code-interpreter wait ci-1a2b3c
grn agentbase tools code-interpreter wait ci-1a2b3c --interval 10s --timeout 30m
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

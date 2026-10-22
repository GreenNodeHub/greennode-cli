# tools code-interpreter session task-wait

Wait for an async command task to finish.

## Description

Poll a `startCommandExecution` task (via `getTask`) until it finishes.

[`session invoke --tool startCommandExecution`](code-interpreter-session-invoke.md)
runs a command **asynchronously**: it returns a `taskId`
(`taskStatus: submitted`) immediately instead of holding the request open.
This command polls `getTask` until `taskStatus` is `completed`, then prints
the final result (`content` plus `structuredContent`: `taskStatus`,
`exitCode`, `startedAt`, `finishedAt`). Progress lines (`Task ... status:
submitted ...`) go to stderr.

It **exits 0 only when the task completed with exit code 0**. A failed task
(the observed lifecycle is `submitted` → `completed`; failed-style statuses
are treated as terminal failures and unknown statuses keep polling), a
non-zero task exit code, or a timeout exits non-zero — after printing the
last `getTask` result, so the task's error output is not lost.

## Synopsis

```text
grn agentbase tools code-interpreter session task-wait <code-interpreter-id> <session-id>
    --task-id <value>
    [--timeout <duration>]
    [--interval <duration>]
    [--end-user-id <value>]
```

## Arguments

**`<code-interpreter-id>`** (string) — code interpreter id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--task-id`** (string) — the task id returned by `startCommandExecution`.
Required.

**`--timeout`** (duration) — maximum time to wait (default `10m`).

**`--interval`** (duration) — poll interval (default `5s`).

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header), 1–128 chars of `A-Za-z0-9._:@-` (validated client-side).

## Output

Mirrors `session invoke`:

- `table` — the result's content items (text items as-is, non-text as JSON),
  then `structuredContent` as indented JSON when present
- `json` — the full final `getTask` response (`content`, `structuredContent`,
  `isError`)
- `id` — the first **text** content item (script-friendly)

## Examples

```bash
# Start an async command (prints a taskId)
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool startCommandExecution --command 'sleep 60 && echo done'

# Wait for it (the command above prints this hint)
grn agentbase tools code-interpreter session task-wait ci-1 cs-1 \
    --task-id task-9f8e7d6c

# Custom poll budget
grn agentbase tools code-interpreter session task-wait ci-1 cs-1 \
    --task-id task-9f8e7d6c --timeout 30m --interval 10s
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`
  (JSON output only), `--endpoint-url`, `--debug`

# tools code-interpreter session invoke

Run a tool inside a code interpreter session.

## Description

Run a built-in tool inside a session
(POST `/v1/code-interpreters/{id}/sessions/{sessionId}/invoke`) and print the
result. The request is **held open server-side** until the command finishes
(the HTTP call blocks; there is no polling).

The tool is chosen with `--tool`; each tool reads its own flags. Required
flags are validated client-side before the request (the API is the source of
truth and rejects bad shapes with a 400 listing the accepted values).

| Tool | Flags |
|---|---|
| `executeCode` | `--language`, `--code` (both required) |
| `executeCommand` | `--command` (required) |
| `startCommandExecution` | `--command` (required; async variant) |
| `listFiles` | `--directory-path` (optional) |
| `writeFiles` | `--content path=content` (the CLI base64-encodes it) or `--content-b64 path=<base64>` (repeatable, at least one) |
| `readFiles` | `--path` (repeatable, at least one) |
| `getTask` | `--task-id` (required) |

`startCommandExecution` is the **async** variant: it returns a `taskId`
(`taskStatus: submitted`) immediately instead of holding the request open.
The CLI prints a hint pointing at
[`session task-wait`](code-interpreter-session-task-wait.md), which polls
`getTask` until the task finishes. `readFiles` returns file contents as
base64 blobs in its content items; `writeFiles` is the inverse — the CLI
base64-encodes `--content` for you.

## Synopsis

```text
grn agentbase tools code-interpreter session invoke <code-interpreter-id> <session-id>
    --tool <value>
    [--language <value>]
    [--code <value>]
    [--command <value>]
    [--directory-path <value>]
    [--content <path=content>]...
    [--content-b64 <path=<base64>>]...
    [--path <value>]...
    [--task-id <value>]
    [--end-user-id <value>]
```

## Arguments

**`<code-interpreter-id>`** (string) — code interpreter id. Required.

**`<session-id>`** (string) — session id. Required.

## Options

**`--tool`** (string) — tool to run. Required; one of `executeCode`,
`executeCommand`, `startCommandExecution`, `listFiles`, `writeFiles`,
`readFiles`, `getTask`.

**`--language`** (string) — `executeCode`: the language (required for
`executeCode`).

**`--code`** (string) — `executeCode`: the code to run (required for
`executeCode`).

**`--command`** (string) — `executeCommand`/`startCommandExecution`: the
command to run (required).

**`--directory-path`** (string) — `listFiles`: directory to list (optional).

**`--content`** (string slice) — `writeFiles`: a file to write as
`path=content` (repeatable; split on the **first** `=`). The content is
**plain text — the CLI base64-encodes it** into the blob the API requires
(use `--content-b64` for data that is already encoded).

**`--content-b64`** (string slice) — `writeFiles`: a file to write as
`path=<base64 blob>` (repeatable; split on the **first** `=`). The blob is
passed through as-is and validated client-side, so a typo fails locally with
a clear message instead of a server 400.

**`--path`** (string slice) — `readFiles`: a file path to read (repeatable).

**`--task-id`** (string) — `getTask`: the task id (required).

**`--end-user-id`** (string) — optional end user id (the `X-End-User-ID`
header), 1–128 chars of `A-Za-z0-9._:@-` (validated client-side).

## Output

- `table` — the result's content items (text items as-is, non-text as JSON),
  then `structuredContent` as indented JSON when present
- `json` — the full response object (`content`, `structuredContent`,
  `isError`)
- `id` — the first **text** content item (script-friendly)

A result with `isError=true` is a **tool-level failure** on an otherwise
successful HTTP call: the CLI prints the result and **exits non-zero**.

## Examples

```bash
# Cheapest deterministic smoke: list the sandbox's files
grn agentbase tools code-interpreter session invoke ci-1 cs-1 --tool listFiles

# Run Python
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool executeCode --language python --code 'print(1+1)'

# Run a shell command
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool executeCommand --command 'uname -a'

# Write then read a file (--content takes plain text; the CLI base64-encodes it)
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool writeFiles --content hello.txt=Hello
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool readFiles --path hello.txt

# writeFiles with pre-encoded data (--content-b64 passes the blob through)
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool writeFiles --content-b64 data.bin=AAECAwQ=

# Start a long-running command asynchronously, then wait for it
grn agentbase tools code-interpreter session invoke ci-1 cs-1 \
    --tool startCommandExecution --command 'sleep 60 && echo done'
# -> prints a taskId, then hints:
#    grn agentbase tools code-interpreter session task-wait ci-1 cs-1 --task-id <id>
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

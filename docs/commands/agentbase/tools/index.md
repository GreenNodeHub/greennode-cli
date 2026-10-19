# `grn agentbase tools`

Manage the **AgentBase built-in tools**: platform-managed headless **browsers**
and **code interpreters** that agents invoke as tools. The group covers the
`agent-core-built-in-tools` service (served at `/built-in-tools` — see
`grn agentbase context current`).

> **Environment note:** the service is **dev-only today**. The prod endpoint
> (`https://agentbase.api.vngcloud.vn/built-in-tools`) follows the shared
> agentbase host pattern and awaits go-live; select dev with
> `grn configure set iam_env dev` (machine) or `grn login --iam-env dev`
> (user).

## Model

Two resources, one shape:

- **`browser`** — a headless browser. Create is asynchronous
  (`CREATING` → `ACTIVE`/`FAILED`): run `browser wait <id>` after create.
  Sessions attach to a browser and expose the CDP **automation stream**, the
  **live-view stream** (screenshots), and the session **recording**.
- **`code-interpreter`** — a code-execution sandbox. Create is asynchronous
  the same way. Sessions are the unit of execution: `session invoke` runs a
  tool (`executeCode`, `executeCommand`, `listFiles`, `writeFiles`,
  `readFiles`, `getTask`, ...) inside a session.

Sessions on both resources accept an optional **`--end-user-id`**
(`X-End-User-ID` header): the end user on whose behalf the session acts
(1–128 chars of `A-Za-z0-9._:@-`, validated client-side).

## Commands

| Command | Description |
|---|---|
| [browser list](browser-list.md) | List browsers |
| [browser create](browser-create.md) | Create a browser (async → `wait`) |
| [browser get](browser-get.md) | Show a browser |
| [browser delete](browser-delete.md) | Delete a browser (and its sessions) |
| [browser wait](browser-wait.md) | Poll a browser until ACTIVE |
| [browser session list](browser-session-list.md) | List a browser's sessions |
| [browser session create](browser-session-create.md) | Create a browser session |
| [browser session get](browser-session-get.md) | Show a session + its streams |
| [browser session delete](browser-session-delete.md) | Delete a session |
| [browser session automation-stream](browser-session-automation-stream.md) | Print the CDP automation stream URL |
| [browser session live-view-stream-sign](browser-session-live-view-stream-sign.md) | Sign the live-view stream URL |
| [browser session recording](browser-session-recording.md) | Download the session recording (NDJSON) |
| [browser session streams update](browser-session-streams-update.md) | Toggle the automation stream |
| [code-interpreter list](code-interpreter-list.md) | List code interpreters |
| [code-interpreter create](code-interpreter-create.md) | Create a code interpreter (async → `wait`) |
| [code-interpreter get](code-interpreter-get.md) | Show a code interpreter |
| [code-interpreter delete](code-interpreter-delete.md) | Delete a code interpreter (and its sessions) |
| [code-interpreter wait](code-interpreter-wait.md) | Poll a code interpreter until ACTIVE |
| [code-interpreter session list](code-interpreter-session-list.md) | List a code interpreter's sessions |
| [code-interpreter session create](code-interpreter-session-create.md) | Create a code interpreter session |
| [code-interpreter session delete](code-interpreter-session-delete.md) | Delete a code interpreter session |
| [code-interpreter session invoke](code-interpreter-session-invoke.md) | Run a tool inside a session |
| [code-interpreter session task-wait](code-interpreter-session-task-wait.md) | Wait for an async command task |

## Lifecycle (browser and code interpreter alike)

```bash
# 1. create (async: returns CREATING)
grn agentbase tools browser create --name my-browser
# 2. wait for ACTIVE (sessions on a non-ACTIVE resource are rejected with 409)
grn agentbase tools browser wait <id>
# 3. use it
grn agentbase tools browser session create <id>
# 4. delete (also deletes sessions; asks for confirmation — see the page)
grn agentbase tools browser delete <id>
```

## API quirks

- **Create is async** on both resources (201 + `CREATING`); session create is
  **synchronous** (the response is the ready session).
- **A session on a non-ACTIVE resource → 409** — always `wait` after create.
- **The recording only exists after the session is deleted** (and archived
  server-side): `session recording` returns 404 until then.
- **Invoke is held open** server-side until the command finishes; a result
  with `isError=true` is a tool-level failure on an otherwise successful HTTP
  call — the CLI prints the result and exits non-zero.
  **`startCommandExecution` is the async exception**: it returns a `taskId`
  immediately — follow it with `code-interpreter session task-wait`.
- Lists use the shared `{listData, page, pageSize, totalItem, totalPage}`
  envelope, 1-based, `--page`/`--size` (validated client-side).

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`
  (JSON output only), `--endpoint-url`, `--debug`

# tools code-interpreter delete

Delete a code interpreter.

## Description

Delete a code interpreter by id (DELETE `/v1/code-interpreters/{id}`).

Deleting a code interpreter **also deletes its sessions**. The delete is
immediate — there is no `--force` and no confirmation prompt.

## Synopsis

```text
grn agentbase tools code-interpreter delete <id>
```

## Arguments

**`<id>`** (string) — code interpreter id. Required.

## Output

- `table` — nothing on stdout (a confirmation line goes to stderr)
- `json` — `{"id": "<id>"}`
- `id` — the deleted code interpreter's id

## Examples

```bash
grn agentbase tools code-interpreter delete ci-1a2b3c
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

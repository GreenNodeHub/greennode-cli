# tools code-interpreter delete

Delete a code interpreter.

## Description

Delete a code interpreter by id (DELETE `/v1/code-interpreters/{id}`).

Deleting a code interpreter **also deletes its sessions** (cascade). The CLI
shows what will be deleted and **asks for confirmation first**: it prints the
code interpreter's id, name, status and session count (a preview + the prompt
go to stderr, so `-o json`/`-o id` stdout stays parseable), then waits for
`y`/`yes`.

- `--force` — skip the prompt (required in non-interactive scripts: without a
  TTY the confirm reads EOF and aborts)
- `--dry-run` — print the preview and exit without deleting anything

## Synopsis

```text
grn agentbase tools code-interpreter delete <id> [--force] [--dry-run]
```

## Arguments

**`<id>`** (string) — code interpreter id. Required.

## Options

**`--force`** (bool) — skip the confirmation prompt.

**`--dry-run`** (bool) — preview what will be deleted without executing.

## Output

- `table` — nothing on stdout (the preview, prompt and confirmation line go
  to stderr)
- `json` — `{"id": "<id>"}`
- `id` — the deleted code interpreter's id

## Examples

```bash
# Preview the cascade without deleting
grn agentbase tools code-interpreter delete ci-1a2b3c --dry-run

# Interactive (asks y/N)
grn agentbase tools code-interpreter delete ci-1a2b3c

# Scripted / CI
grn agentbase tools code-interpreter delete ci-1a2b3c --force
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

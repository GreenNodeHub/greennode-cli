# tools browser delete

Delete a headless browser.

## Description

Delete a browser by id (DELETE `/v1/browsers/{id}`).

Deleting a browser **also deletes its sessions**. The delete is immediate —
there is no `--force` and no confirmation prompt.

## Synopsis

```text
grn agentbase tools browser delete <id>
```

## Arguments

**`<id>`** (string) — browser id. Required.

## Output

- `table` — nothing on stdout (a confirmation line goes to stderr)
- `json` — `{"id": "<id>"}`
- `id` — the deleted browser's id

## Examples

```bash
grn agentbase tools browser delete br-1a2b3c
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

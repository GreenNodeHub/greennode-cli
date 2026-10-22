# tools browser get

Show a headless browser.

## Description

Show one browser by id (GET `/v1/browsers/{id}`).

## Synopsis

```text
grn agentbase tools browser get <id>
```

## Arguments

**`<id>`** (string) — browser id. Required.

## Output

- `table` — field/value (ID, Name, Description, Status, Created, Updated)
- `json` — the raw browser object
- `id` — the browser's id

## Examples

```bash
grn agentbase tools browser get br-1a2b3c
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

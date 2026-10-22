# tools code-interpreter get

Show a code interpreter.

## Description

Show one code interpreter by id (GET `/v1/code-interpreters/{id}`).

## Synopsis

```text
grn agentbase tools code-interpreter get <id>
```

## Arguments

**`<id>`** (string) — code interpreter id. Required.

## Output

- `table` — field/value (ID, Name, Description, Status, Created, Updated)
- `json` — the raw code interpreter object
- `id` — the code interpreter's id

## Examples

```bash
grn agentbase tools code-interpreter get ci-1a2b3c
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

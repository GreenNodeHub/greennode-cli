# runtime delete

Delete an agent runtime.

## Description

Delete an agent runtime, addressed by its `id`. Deletion is asynchronous; this
command returns as soon as the service accepts the request. Confirm with
`grn agentbase runtime wait <id>` (it reaches `DELETED` on success).

## Synopsis

```text
grn agentbase runtime delete <id>
```

## Options

This command takes no command-specific options. The runtime `id` is supplied as
the single positional argument.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Delete a runtime, then wait for it to reach `DELETED`:

```bash
grn agentbase runtime delete agn-rt-abc123
grn agentbase runtime wait agn-rt-abc123
```

Print only the deleted id (for scripting):

```bash
grn agentbase runtime delete agn-rt-abc123 -o id
```

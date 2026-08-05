# runtime get

Show an agent runtime.

## Description

Show an agent runtime, addressed by its `id` (the `name` is immutable and not
addressable). Prints the runtime's `ID`, `Name`, `Description`, `State`, `Status
Reason`, `Created`, and `Updated`.

## Synopsis

```text
grn agentbase runtime get <id>
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

Show a runtime in table form:

```bash
grn agentbase runtime get agn-rt-abc123
```

Show the raw JSON response:

```bash
grn agentbase runtime get agn-rt-abc123 -o json
```

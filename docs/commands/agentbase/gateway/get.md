# gateway get

Show a gateway.

## Description

Fetch and display a single gateway by name, rendered as a detail table (or
JSON/id per the global output format).

## Synopsis

```text
grn agentbase gateway get <name>
```

## Arguments

**`<name>`** (string)

Name of the gateway to show.

- Required: Yes (exactly one positional argument)

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Show a gateway:

```bash
grn agentbase gateway get my-gateway
```

Show a gateway as JSON:

```bash
grn agentbase gateway get my-gateway -o json
```

# deploy generate

Print an agent manifest template (YAML or JSON).

## Description

Print a commented agent manifest to stdout. Save it, fill it in, and apply with `grn agentbase deploy up --file <file>`.

Defaults to YAML (with comments); pass `-o json` for a JSON skeleton.

## Synopsis

```text
grn agentbase deploy generate
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`). With `generate`, `json` selects the JSON skeleton instead of the commented YAML template.
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Print the YAML template and save it:

```bash
grn agentbase deploy generate > agent.yaml
```

Print a JSON skeleton instead:

```bash
grn agentbase deploy generate -o json
```

# memory generate

Print a memory create template (YAML or JSON).

## Description

Print a commented memory create template to stdout. Save it, fill it in, and apply with `grn agentbase memory create --file <file>`.

Defaults to YAML (with comments); pass `-o json` for a JSON skeleton.

## Synopsis

```text
grn agentbase memory generate
```

## Options

This command takes no command-specific options. Use the global `-o, --output` to select the template format: `table`/omitted prints commented YAML, `json` prints a JSON skeleton.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`); for `generate`, `json` selects the JSON skeleton.
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Print the commented YAML template to a file:

```bash
grn agentbase memory generate > mem.yaml
```

Print a JSON skeleton instead:

```bash
grn agentbase memory generate -o json
```

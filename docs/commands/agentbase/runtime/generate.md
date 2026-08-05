# runtime generate

Print a runtime create template (YAML or JSON).

## Description

Print a commented agent-runtime create template to stdout. Save it, fill it in,
and apply with `grn agentbase runtime create --file <file>`.

Defaults to YAML (with comments); pass `-o json` for a JSON skeleton.

## Synopsis

```text
grn agentbase runtime generate
```

## Options

This command takes no command-specific options. Use the global `-o, --output`
flag to switch the emitted template format: default `table` (YAML template) or
`json` (JSON skeleton).

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Print the default YAML template and save it:

```bash
grn agentbase runtime generate > rt.yaml
```

Print a JSON skeleton:

```bash
grn agentbase runtime generate -o json
```

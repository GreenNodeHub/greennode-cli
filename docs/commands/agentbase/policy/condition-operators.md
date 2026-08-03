# policy condition-operators

List accepted policy condition operators.

## Description

List the condition operators accepted by a policy statement's `condition`
branch (the `when`/`unless` maps). Each entry reports the operator name, arity,
accepted value types, and display name. This is a read-only catalog; use it when
authoring a `condition` block in a policy spec file.

## Synopsis

```text
grn agentbase policy condition-operators
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

List the operators (table):

```bash
grn agentbase policy condition-operators
```

Emit the full catalog as JSON:

```bash
grn agentbase policy condition-operators -o json
```

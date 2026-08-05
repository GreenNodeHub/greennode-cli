# cr registry-credential reset-secret

Rotate the robot-account secret.

## Description

Rotate the robot-account secret. The previous secret is invalidated immediately
— any CI or local credentials must be updated. The new secret is MASKED in
table output (last-4 shown); use `-o json` to reveal it.

## Synopsis

```text
grn agentbase cr registry-credential reset-secret
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase cr registry-credential reset-secret
```

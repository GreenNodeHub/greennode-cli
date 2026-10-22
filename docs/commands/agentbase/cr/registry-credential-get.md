# cr registry-credential get

Show the robot account (username + secret).

## Description

Fetch the registry robot account for push/pull. Its secret is `[REDACTED]` in every format unless `--show-secret` is explicit.

## Synopsis

```text
grn agentbase cr registry-credential get
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
grn agentbase cr registry-credential get
```

Reveal the full secret for `docker login`:

```bash
grn agentbase cr registry-credential get -o json --show-secret
```

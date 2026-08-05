# cr registry-credential get

Show the robot account (username + secret).

## Description

Fetch the robot account (username + secret) used to authenticate to the
registry for `docker login` push/pull. The secret is MASKED in table output
(only the last 4 characters are shown); use `-o json` to reveal the full
secret so it can be piped into `docker login`.

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
grn agentbase cr registry-credential get -o json
```

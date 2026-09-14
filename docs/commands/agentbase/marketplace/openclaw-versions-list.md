# marketplace openclaw-versions list

List OpenClaw versions.

## Description

List the OpenClaw versions served by the runtime service catalog
(GET `/v1/openclaw-versions`). Each version carries an id, name, and whether
it is the default deployment.

## Synopsis

```text
grn agentbase marketplace openclaw-versions list
```

## Options

This command takes no command-specific options.

## Examples

```bash
grn agentbase marketplace openclaw-versions list
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

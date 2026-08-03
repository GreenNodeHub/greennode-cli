# cr repository get

Show the user's repository info.

## Description

Fetch and display the user's auto-provisioned repository: name, registry URL,
image count, and quota usage. The repository and robot account are created on
first access if they do not yet exist — there is no `create` step.

## Synopsis

```text
grn agentbase cr repository get
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
grn agentbase cr repository get
```

Show the repository as JSON (e.g. to read the registry URL for `docker login`):

```bash
grn agentbase cr repository get -o json
```

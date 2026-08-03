# policy group update

Update a policy group.

## Description

Update a policy group's `name` and/or `description`. Individual flags are
applied as a merge-patch — only fields explicitly set on the command line are
sent. Alternatively, `--file` provides a full replacement spec (see
[group generate](group-generate.md)) and overrides the flags.

## Synopsis

```text
grn agentbase policy group update <group-id>
    [--name <value>]
    [--description <value>]
    [--file <value>]
```

Positional arguments:

- `<group-id>` — the policy group id (exactly 1 required).

## Options

**`--name`** (string)

New name (only applied when set).

- Required: No

**`--description`** (string)

New description (only applied when set).

- Required: No

**`--file`** (string)

Replacement spec (see [group generate](group-generate.md)).

- Required: No
- Authoritative when set: sends `name` and `description` from the file (full replacement).

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Rename a group (merge-patch):

```bash
grn agentbase policy group update pg-abc123 --name prod-group
```

Replace from a spec file:

```bash
grn agentbase policy group update pg-abc123 --file group.yaml
```

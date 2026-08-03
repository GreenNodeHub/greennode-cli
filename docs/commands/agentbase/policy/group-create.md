# policy group create

Create a policy group.

## Description

Create a new policy group owned by the authenticated user. A group is the
container of policies that a gateway binds via its `policyGroupId` (max 20 groups
per user).

Provide `--name` (and optionally `--description`), or apply a full spec with
`--file`. The spec file (see [group generate](group-generate.md)) is a YAML or
JSON document with `name` (required) and `description` keys; `--file` is
authoritative when set.

## Synopsis

```text
grn agentbase policy group create
    --name <value>
    [--description <value>]
    [--file <value>]
```

## Options

**`--name`** (string)

Group name (required without `--interactive`). Unique per user.

- Required: Yes (without `--file`)
- Alias: `-n`

**`--description`** (string)

Description.

- Required: No

**`--file`** (string)

Apply a spec file (see [group generate](group-generate.md)).

- Required: No
- Authoritative when set: `--name`/`--description` flags are ignored.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Create a group from flags:

```bash
grn agentbase policy group create --name my-group
```

Create a group from a spec file:

```bash
grn agentbase policy group create --file group.yaml
```

Prompt for the missing name:

```bash
grn agentbase policy group create --interactive --description "A group of authorization policies"
```

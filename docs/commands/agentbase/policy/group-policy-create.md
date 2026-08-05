# policy group policy create

Create a policy within a group.

## Description

Create a new policy (a permit/forbid rule) inside a policy group (max 10
policies per group). The policy's `statement` is a `PolicyTemplate` that the
server compiles to Cedar at write time — an invalid statement is rejected with
HTTP 400 before being stored.

Provide the simple flag path (`--name` + the statement fields
`--effect`/`--principal`/`--action`/`--resource`) or apply a full spec with
`--file`. `--file` is authoritative when set. The `condition` branch is **only**
expressible via `--file` (see [group policy generate](group-policy-generate.md)).

## Synopsis

```text
grn agentbase policy group policy create <group-id>
    --name <value>
    --effect <value>
    --principal <value>
    --action <value>
    [--description <value>]
    [--active]
    [--resource <value>]
    [--file <value>]
```

Positional arguments:

- `<group-id>` — the policy group id (exactly 1 required).

## Options

**`--name`** (string)

Policy name (required without `--interactive`).

- Required: Yes (without `--file`)
- Alias: `-n`

**`--description`** (string)

Description.

- Required: No

**`--active`** (boolean)

Whether the policy is active.

- Required: No
- Default: `false`

**`--effect`** (string)

Statement effect (permit|forbid).

- Required: Yes (without `--file`)
- Possible values: `permit`, `forbid`

**`--principal`** (string)

Principal entity id (e.g. `jwt_user:abc-123`, `iam_role:admin`, or `*`).

- Required: Yes (without `--file`)

**`--action`** (list&lt;string&gt;)

Action name (repeat, or `*`); e.g. `InsuranceAPI__read`.

- Required: Yes (without `--file`)
- Repeatable: Yes — specify once per value.
- Constraint: action names match `^[A-Za-z0-9_]+__[A-Za-z0-9_]+$`, or `*`.

**`--resource`** (list&lt;string&gt;)

Resource gateway ref (repeat); e.g. `gateway:my-gw` or `gateway:*`.

- Required: No
- Repeatable: Yes — specify once per value.

**`--file`** (string)

Apply a spec file (see [group policy generate](group-policy-generate.md));
authoritative when set.

- Required: No
- Authoritative when set: flags are ignored and the full spec (including
  optional `statement.condition`) is applied.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Create a policy from flags:

```bash
grn agentbase policy group policy create pg-abc123 \
  --name allow-admin-read \
  --effect permit \
  --principal "jwt_role:admin" \
  --action InsuranceAPI__read \
  --resource "gateway:*" \
  --active
```

Create a policy from a spec file (the only way to add a `condition`):

```bash
grn agentbase policy group policy create pg-abc123 --file policy.yaml
```

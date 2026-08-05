# policy group policy update

Update a policy (merge-patch).

## Description

Update a policy within a group. Individual flags are applied as a merge-patch:
`--name`, `--description`, and `--active` are sent only when set; any of the
statement fields (`--effect`, `--principal`, `--action`, `--resource`) being set
replaces the whole `statement`.

Alternatively, `--file` provides a full replacement spec (see
[group policy generate](group-policy-generate.md)) and sends
name/description/statement/active together. As with create, the `statement` is
compiled to Cedar at write time — an invalid statement is rejected with HTTP 400.

## Synopsis

```text
grn agentbase policy group policy update <group-id> <policy-id>
    [--name <value>]
    [--description <value>]
    [--active]
    [--effect <value>]
    [--principal <value>]
    [--action <value>]
    [--resource <value>]
    [--file <value>]
```

Positional arguments:

- `<group-id>` — the policy group id (exactly 1 required).
- `<policy-id>` — the policy id (exactly 1 required).

## Options

**`--name`** (string)

New name (only applied when set).

- Required: No

**`--description`** (string)

New description (only applied when set).

- Required: No

**`--active`** (boolean)

Activate/deactivate (only applied when set).

- Required: No
- Default: `false`

**`--effect`** (string)

Statement effect (permit|forbid).

- Required: No
- Possible values: `permit`, `forbid`
- Replaces the entire `statement` when any statement flag is set.

**`--principal`** (string)

Principal entity id (e.g. `jwt_user:abc-123`, `iam_role:admin`, or `*`).

- Required: No
- Replaces the entire `statement` when any statement flag is set.

**`--action`** (list&lt;string&gt;)

Action name (repeat, or `*`); e.g. `InsuranceAPI__read`.

- Required: No
- Repeatable: Yes — specify once per value.
- Replaces the entire `statement` when any statement flag is set.

**`--resource`** (list&lt;string&gt;)

Resource gateway ref (repeat); e.g. `gateway:my-gw` or `gateway:*`.

- Required: No
- Repeatable: Yes — specify once per value.
- Replaces the entire `statement` when any statement flag is set.

**`--file`** (string)

Full replacement spec (see [group policy generate](group-policy-generate.md)).

- Required: No
- Authoritative when set: sends name/description/statement/active together
  (full replacement).

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Rename and activate a policy (merge-patch):

```bash
grn agentbase policy group policy update pg-abc123 pol-def456 --name allow-admin --active
```

Replace the whole policy from a spec file:

```bash
grn agentbase policy group policy update pg-abc123 pol-def456 --file policy.yaml
```

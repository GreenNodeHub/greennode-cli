# gateway update

Update a gateway's mutable fields.

## Description

Update a gateway's mutable fields (JSON Merge Patch semantics: omit = leave
alone, `null` = clear, value = replace). Sealed fields (`name`, `networkMode`,
`flavorId`, `replicas`, `privateNetwork.vpcId`/`subnetId`) cannot be changed —
recreate the gateway instead.

Flags set only the simple mutable fields. For `inboundAuth`, `targets`,
`hostAliases`, or `privateNetwork.routes` use `--file` with a partial merge-patch
(template: `grn agentbase gateway generate`, then keep only the keys to change).

Clear the policy group with `--clear-policy-group-id`.

The gateway is updated asynchronously; this command returns as soon as the
service accepts the patch. Converge with `grn agentbase gateway wait <name>`.

## Synopsis

```text
grn agentbase gateway update <name>
    [--file <value>]
    [--display-name <value>]
    [--description <value>]
    [--policy-group-id <value>]
    [--clear-policy-group-id]
    [--allowed-cidr <value>]
    [--host-alias <value>]
```

## Arguments

**`<name>`** (string)

Name of the gateway to update.

- Required: Yes (exactly one positional argument)

## Options

**`--file`** (string)

Apply a partial merge-patch spec file (YAML or JSON). Authoritative when set —
the individual flags below are ignored.

- Required: No

**`--display-name`** (string)

Set the display name.

- Required: No

**`--description`** (string)

Set the description.

- Required: No

**`--policy-group-id`** (string)

Set the policy group id binding.

- Required: No

**`--clear-policy-group-id`** (boolean)

Clear the policy group binding (sends `policyGroupId: null`).

- Required: No
- Default: `false`

**`--allowed-cidr`** (list&lt;string&gt;)

Replace the inbound allowlist (IPv4 CIDRs). Omit to leave unchanged. *(repeatable)*

- Required: No

**`--host-alias`** (list&lt;string&gt;)

Replace the `/etc/hosts` overrides, in `ip=host1,host2` form. *(repeatable)*

- Required: No
- Syntax: `--host-alias 10.0.0.1=foo.local,bar.local`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Update the display name and description from flags:

```bash
grn agentbase gateway update my-gateway \
  --display-name "My Gateway" \
  --description "Updated description"
grn agentbase gateway wait my-gateway
```

Clear the policy group binding:

```bash
grn agentbase gateway update my-gateway --clear-policy-group-id
```

Apply a partial merge-patch from a file (for nested fields):

```bash
grn agentbase gateway update my-gateway --file patch.yaml
```

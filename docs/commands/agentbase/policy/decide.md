# policy decide

Probe an authorization decision for a gateway target.

## Description

Probe an authorization decision — the **internal** route that agent-core-gateway
calls per inbound request. It always returns a decision: `allow`, or `deny` with
a `reason` (HTTP 200 in both cases).

The minimal flag path covers the common probe (policy group + user + action
method). For richer inputs (principal attributes, `context`, or
`action.params.arguments`), pass the full `DecisionRequest` body via `--file`,
crafted from the API docs.

## Synopsis

```text
grn agentbase policy decide <gateway> <target>
    --policy-group-id <value>
    --method <value>
    [--user-id <value>]
    [--user-type <value>]
    [--action-name <value>]
    [--jsonrpc <value>]
    [--file <value>]
```

Positional arguments:

- `<gateway>` — the gateway id (exactly 1 required).
- `<target>` — the target id (exactly 1 required).

## Options

**`--policy-group-id`** (string)

Policy group id (required without `--file`).

- Required: Yes (without `--file`)

**`--user-id`** (string)

End user id being evaluated.

- Required: No

**`--user-type`** (string)

End user type (iam|jwt).

- Required: No
- Possible values: `iam`, `jwt`

**`--method`** (string)

JSON-RPC action method (required without `--file`), e.g. `tools/call`.

- Required: Yes (without `--file`)

**`--action-name`** (string)

JSON-RPC `params.name` (effective action); optional.

- Required: No

**`--jsonrpc`** (string)

JSON-RPC version.

- Required: No
- Default: `2.0`

**`--file`** (string)

Full `DecisionRequest` body (for principal/context/arguments).

- Required: No
- Authoritative when set: flags are ignored and the full body is applied. The
  spec must include `policyGroupId` and `action.method`.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Probe the common case (policy group + user + method):

```bash
grn agentbase policy decide gw-abc tg-def \
  --policy-group-id pg-123 \
  --user-id u-456 \
  --user-type jwt \
  --method tools/call \
  --action-name InsuranceAPI__read
```

Probe with a full decision body from a file:

```bash
grn agentbase policy decide gw-abc tg-def --file decision.json
```

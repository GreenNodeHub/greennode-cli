# identity workload update

Update an agent identity.

## Description

Update an existing agent identity. Only description and allowed return URLs can be modified.

## Synopsis

```text
grn agentbase identity workload update <name>
    [--description <value>]
    [--allowed-return-url <value>]
```

## Options

**`--description`** (string)

Updated description.

- Required: No
- Default: _(empty)_

**`--allowed-return-url`** (list&lt;string&gt;)

Allowed return URL (repeatable). Replaces the existing list when provided.

- Required: No
- Default: _(none)_
- Syntax: repeat the flag, e.g. `--allowed-return-url https://a.example --allowed-return-url https://b.example`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Update the description:

```bash
grn agentbase identity workload update my-agent --description "Updated description"
```

Replace the allowed return URLs:

```bash
grn agentbase identity workload update my-agent \
  --allowed-return-url https://app.example.com/callback \
  --allowed-return-url https://app.example.com/alt
```

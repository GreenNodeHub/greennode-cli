# access outbound-auth delegated get-key

Obtain a delegated API key for an agent identity.

## Description

Obtain a delegated API key for an agent identity from a delegated API key provider.

Required flags: `--agent-user-id` and `--return-url`.
Optional flags: `--custom-state`, `--session-id`, `--force-delegation`.

The response may include the API key value and an authorization URL. Use `-o json`
(or `-o id`) to reveal the full values; the default `table` output is for human inspection.

## Synopsis

```text
grn agentbase access outbound-auth delegated get-key <provider-name> <identity-name>
    --agent-user-id <value>
    --return-url <value>
    [--custom-state <value>]
    [--session-id <value>]
    [--force-delegation]
```

## Arguments

- `<provider-name>` — name of the delegated API key provider
- `<identity-name>` — name of the agent identity

## Options

**`--agent-user-id`** (string)

Agent user ID (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

**`--return-url`** (string)

Return URL after authorization (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)

**`--custom-state`** (string)

Custom state parameter.

- Required: No
- Default: _(empty)_

**`--session-id`** (string)

Session ID (UUID format).

- Required: No
- Default: _(empty)_

**`--force-delegation`** (boolean)

Force delegation.

- Required: No
- Default: `false`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Obtain a delegated API key:

```bash
grn agentbase access outbound-auth delegated get-key \
  my-delegated-provider my-agent \
  --agent-user-id user-123 \
  --return-url https://app.example.com/callback
```

Force a fresh delegation and reveal the full response as JSON (prints secret values):

```bash
grn agentbase access outbound-auth delegated get-key \
  my-delegated-provider my-agent \
  --agent-user-id user-123 \
  --return-url https://app.example.com/callback \
  --force-delegation \
  -o json
```

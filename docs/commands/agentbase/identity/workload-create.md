# identity workload create

Create a new agent identity.

## Description

Create a new agent identity for the authenticated user.

Agent identities are used to represent digital identities for agents accessing
external services. The name must be 3-50 characters and match the pattern
`^[a-zA-Z0-9_-]+$`.

## Synopsis

```text
grn agentbase identity workload create
    --name <value>
    [--set-current]
    [--description <value>]
    [--allowed-return-url <value>]
```

## Options

**`--name`** (string)

Agent identity name (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)
- Alias: `-n`
- Constraints: 3–50 characters; matches `^[a-zA-Z0-9_-]+$`.

**`--set-current`** (boolean)

Set as the current agent identity after creation (persists to the shared `~/.greennode` profile, per-profile).

- Required: No
- Default: `false`

**`--description`** (string)

Description of the agent identity.

- Required: No
- Default: _(empty)_

**`--allowed-return-url`** (list&lt;string&gt;)

Allowed return URL (repeatable).

- Required: No
- Default: _(none)_
- Syntax: repeat the flag, e.g. `--allowed-return-url https://a.example --allowed-return-url https://b.example`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Create an agent identity:

```bash
grn agentbase identity workload create --name my-agent
```

Create one with a description, allowed return URLs, and set it as current:

```bash
grn agentbase identity workload create \
  --name my-agent \
  --description "Production agent" \
  --allowed-return-url https://app.example.com/callback \
  --set-current
```

Create interactively (prompts for the name):

```bash
grn agentbase identity workload create --interactive
```

# identity outbound-auth oauth2 m2m-token

Get an M2M OAuth2 token.

## Description

Retrieve a machine-to-machine (client credentials) OAuth2 token for an agent identity via an OAuth2 provider.

This returns the access token. Use `-o json` (or `-o id`) to reveal the full
value; the default `table` output is for human inspection.

## Synopsis

```text
grn agentbase identity outbound-auth oauth2 m2m-token <provider-name> <identity-name>
    --scope <value>
```

## Arguments

- `<provider-name>` — name of the OAuth2 provider
- `<identity-name>` — name of the agent identity

## Options

**`--scope`** (list&lt;string&gt;)

OAuth2 scope (repeatable, required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted as a comma-separated list when `--interactive` is set)
- Syntax: repeat the flag, e.g. `--scope read --scope write`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get an M2M token with scopes (table view):

```bash
grn agentbase identity outbound-auth oauth2 m2m-token \
  my-oauth2-provider my-agent \
  --scope read --scope write
```

Reveal the full access token value (use with care — this prints the secret):

```bash
grn agentbase identity outbound-auth oauth2 m2m-token \
  my-oauth2-provider my-agent \
  --scope read \
  -o json
```

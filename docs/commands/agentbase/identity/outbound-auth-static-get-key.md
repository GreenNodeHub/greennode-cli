# identity outbound-auth static get-key

Get the API key for an agent identity.

## Description

Retrieve the API key assigned to a specific agent identity from a static API key provider.

This returns the secret API key value. Use `-o json` (or `-o id`) to reveal the full
value; the default `table` output is for human inspection.

## Synopsis

```text
grn agentbase identity outbound-auth static get-key <provider-name> <identity-name>
```

## Arguments

- `<provider-name>` — name of the static API key provider
- `<identity-name>` — name of the agent identity

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Get the API key assigned to an identity (table view):

```bash
grn agentbase identity outbound-auth static get-key my-apikey-provider my-agent
```

Reveal the full API key value (use with care — this prints the secret):

```bash
grn agentbase identity outbound-auth static get-key my-apikey-provider my-agent -o json
```

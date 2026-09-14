# access outbound-auth static create

Create a static API key provider.

## Description

Create a new static API key provider. The name must be 3-50 characters and match
the pattern `^[a-zA-Z0-9_-]+$`. Both name and API key are required.

## Synopsis

```text
grn agentbase access outbound-auth static create
    --name <value>
    --apikey <value>
```

## Options

**`--name`** (string)

Provider name (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted when `--interactive` is set)
- Alias: `-n`
- Constraints: 3–50 characters; matches `^[a-zA-Z0-9_-]+$`.

**`--apikey`** (string)

API key value (required without `--interactive`).

- Required: Yes (enforced in non-interactive mode; prompted as a secret when `--interactive` is set)

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`; shadows `grn`'s inherited `--output`)
- `-i, --interactive` — prompt for missing inputs instead of requiring flags
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`, …

## Examples

Create a static API key provider:

```bash
grn agentbase access outbound-auth static create \
  --name my-apikey-provider \
  --apikey sk-xxxxxxxxxxxx
```

Create interactively (the API key is read without echoing):

```bash
grn agentbase access outbound-auth static create --name my-apikey-provider --interactive
```

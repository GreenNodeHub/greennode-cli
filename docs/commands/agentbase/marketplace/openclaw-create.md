# marketplace openclaw create

Create an OpenClaw workspace.

## Description

Create an OpenClaw workspace (POST `/v1/openclaws`). The creator spec carries
the model provider credentials, environment variables, messaging channels
(Telegram, Zalo), flavor, and a proof-of-concept flag. For anything beyond the
simple `--name/--version-id/--flavor-id` path, author a spec file with
`--file` (authoritative when set).

## Synopsis

```text
grn agentbase marketplace openclaw create
    [--name <value>]
    [--version-id <value>]
    [--flavor-id <value>]
    [--poc]
    [--file <path>]
```

## Options

**`--name`** (string)

OpenClaw name (required without `--file`).

- Required: No (required without `--file` / `--interactive`)

**`--version-id`** (string)

OpenClaw version id (required without `--file`).

- Required: No (required without `--file` / `--interactive`)

**`--flavor-id`** (string)

Flavor id (required without `--file`).

- Required: No (required without `--file` / `--interactive`)

**`--poc`** (boolean)

Mark as proof-of-concept.

- Required: No
- Default: `false`

**`--file`** (string)

Apply a spec file (authoritative when set).

- Required: No

## Examples

Create from flags:

```bash
grn agentbase marketplace openclaw create --name my-claw --version-id v1 --flavor-id f1
```

Create from a spec file:

```bash
grn agentbase marketplace openclaw create --file openclaw.yaml
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

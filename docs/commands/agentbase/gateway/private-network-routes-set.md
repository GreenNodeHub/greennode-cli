# gateway private-network routes set

Replace a PRIVATE-mode gateway's private-network routes.

## Description

Replace (PUT, full replacement) the CIDR routes on a PRIVATE-mode gateway's
worker nodes (PUT `/api/v1/gateways/{name}/private-network/routes`). Pass
`--route` repeatedly, or `--file` with a `{routes: [...]}` JSON/YAML document.
`--if-match` is the optional ETag (from a prior get) for optimistic
concurrency; omit to force the replace.

## Synopsis

```text
grn agentbase gateway private-network routes set <name>
    [--route <cidr> ...]
    [--file <path>]
    [--if-match <etag>]
```

## Arguments

**`<name>`** (string) — gateway name. Required.

## Options

**`--route`** (string, repeatable) — CIDR route (ignored with `--file`).
**`--file`** (string) — JSON/YAML `{routes: [...]}` spec (authoritative when set).
**`--if-match`** (string) — If-Match ETag for optimistic concurrency (optional).

## Examples

```bash
grn agentbase gateway private-network routes set my-gw --route 172.16.0.0/12 --route 10.0.0.0/16
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

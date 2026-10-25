# runtime update

Update an agent runtime (full-spec replacement).

## Description

Update an agent runtime. Unlike gateway, this is a FULL-SPEC replacement, not a
merge-patch: every field is required server-side (the create spec minus `name`).
Updating creates a new version and rolls the default endpoint forward.

For anything beyond the simple path, use `--file` with the create template
([generate](generate.md)) minus the `name` field.

When `--file` is set it is authoritative — every flag below is ignored. The file
is validated as a full create spec, so `name` must be present (and non-empty) in
the file even though it is not sent; the runtime is addressed by the `<id>`
positional argument.

> **Notes:** `--min-replicas` is gone (the platform owns the autoscaling floor —
> always 1); only `--max-replicas` is configurable. `--poc` is create-only and
> cannot be set or cleared on update.

## Synopsis

```text
grn agentbase runtime update <id>
    --image-url <value>
    --flavor-id <value>
    [--description <value>]
    [--image-auth-enabled]
    [--image-auth-username <value>]
    [--image-auth-password <value>]
    [--image-auth-use-registry-credentials]
    [--command <value>]...
    [--args <value>]...
    [--env KEY=VALUE]...
    [--max-replicas <value>]
    [--cpu-utilization <value>]
    [--memory-utilization <value>]
    [--network-mode <value>]
    [--vpc-id <value>]
    [--subnet-id <value>]
    [--route-cidrs <value>]...
    [--allowed-cidrs <value>]...
    [--inbound-mode <value>]
    [--inbound-jwt-source <value>]
    [--inbound-jwt-discovery-url <value>]
    [--inbound-jwt-jwks <value>]
    [--inbound-jwt-audience <value>]...
    [--inbound-jwt-client <value>]...
    [--inbound-jwt-scope <value>]...
    [--inbound-jwt-principal-claim <value>]
    [--protocol <value>]
    [--file <spec.yaml>]
```

## Options

**`--image-url`** (string)

Container image URL.

- Required: Yes
- Default: (empty)

**`--flavor-id`** (string)

Flavor id.

- Required: Yes
- Default: (empty)

**`--description`** (string)

Description.

- Required: No
- Default: (empty)

**`--image-auth-enabled`** (bool)

Enable private-registry auth (requires `--image-auth-username` /
`--image-auth-password`).

- Required: No
- Default: `false`

**`--image-auth-username`** (string)

Private-registry username (used with `--image-auth-enabled`).

- Required: Conditional — required when image auth is active (`--image-auth-enabled` set, or a username is supplied).
- Default: (empty)

**`--image-auth-password`** (string)

Private-registry password (used with `--image-auth-enabled`).

- Required: Conditional — required when image auth is active.
- Default: (empty)
- Constraints: send-only — never present on responses. Prefer supplying it via `--file` rather than on the command line.

**`--image-auth-use-registry-credentials`** (bool)

`imageAuth`: let the platform pull the image using its own registry
credentials. Mutually exclusive with username/password auth.

- Required: No
- Default: `false`

**`--command`** (stringArray)

Container entrypoint element (repeatable).

- Required: No
- Default: (empty)
- Syntax: `--command python --command -m --command my_agent`

**`--args`** (stringArray)

Container arg (repeatable).

- Required: No
- Default: (empty)
- Syntax: `--args --debug --args --port=8080`

**`--env`** (stringArray)

Environment variable `KEY=VALUE` (repeatable).

- Required: No
- Default: (empty)
- Syntax: `--env LOG_LEVEL=info --env DEBUG=1`

**`--max-replicas`** (int)

Autoscaling: maximum replicas (1-10). The minimum is platform-owned (always 1).

- Required: No
- Default: `2`
- Constraints: 1–10.

**`--cpu-utilization`** (int)

Autoscaling: CPU scale-up threshold % (10-90).

- Required: No
- Default: `70`
- Constraints: 10–90.

**`--memory-utilization`** (int)

Autoscaling: memory scale-up threshold % (10-90).

- Required: No
- Default: `70`
- Constraints: 10–90.

**`--network-mode`** (string)

`networkConfig` mode: `PUBLIC` or `VPC` (unset = `PUBLIC`).

- Required: No
- Default: `PUBLIC`
- Constraints: `VPC` requires `--vpc-id` and `--subnet-id`.

**`--vpc-id`** (string)

`networkConfig` VPC id (required with `--network-mode VPC`).

- Required: Conditional — required when `--network-mode VPC`.
- Default: (empty)

**`--subnet-id`** (string)

`networkConfig` subnet id (required with `--network-mode VPC`).

- Required: Conditional — required when `--network-mode VPC`.
- Default: (empty)

**`--route-cidrs`** (stringArray)

`networkConfig` egress route CIDR (repeatable, max 20, VPC only).

- Required: No
- Default: (empty)
- Syntax: `--route-cidrs 10.0.0.0/16 --route-cidrs 172.16.0.0/12`

**`--allowed-cidrs`** (stringArray)

Inbound CIDR allowlist (repeatable, max 20).

- Required: No
- Default: (empty)
- Syntax: `--allowed-cidrs 203.0.113.0/24`

**`--inbound-mode`** (string)

`inboundAuth` mode: `NONE`, `IAM`, or `JWT` (unset = `NONE`).

- Required: No
- Default: (empty = `NONE`)
- Constraints: `JWT` requires the `--inbound-jwt-*` flags; cannot be combined
  with them when set to `IAM`/`NONE`.

**`--inbound-jwt-source`** (string)

`inboundAuth` JWT key source: `DISCOVERY` or `JWKS` (required with
`--inbound-mode JWT`).

- Required: Conditional — required when `--inbound-mode JWT`.
- Default: (empty)

**`--inbound-jwt-discovery-url`** (string)

`inboundAuth` JWT OIDC discovery URL (required with source `DISCOVERY`).

- Required: Conditional — required when `--inbound-jwt-source DISCOVERY`.
- Default: (empty)

**`--inbound-jwt-jwks`** (string)

`inboundAuth` JWT inline key set JSON (required with source `JWKS`, max 32KiB).
Prefer `--file` — the value is a full JSON document.

- Required: Conditional — required when `--inbound-jwt-source JWKS`.
- Default: (empty)

**`--inbound-jwt-audience`** (stringArray)

`inboundAuth` JWT allowed audience (repeatable, max 20).

- Required: No
- Default: (empty)

**`--inbound-jwt-client`** (stringArray)

`inboundAuth` JWT allowed client (repeatable, max 20).

- Required: No
- Default: (empty)

**`--inbound-jwt-scope`** (stringArray)

`inboundAuth` JWT allowed scope (repeatable, max 50).

- Required: No
- Default: (empty)

**`--inbound-jwt-principal-claim`** (string)

`inboundAuth` JWT principal claim (default `sub`).

- Required: No
- Default: `sub`

**`--protocol`** (string)

Runtime protocol: `HTTP` or `MCP` (unset = `HTTP`).

- Required: No
- Default: (empty = `HTTP`)

**`--file`** (string)

Apply a full-spec file (create template minus `name`); authoritative when set.

- Required: No
- Default: (empty)
- Constraints: when set, the flags above are ignored; `name` must be present in the file.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Update from a full-spec file (recommended):

```bash
grn agentbase runtime generate > rt.yaml
# ...edit rt.yaml (keep name)...
grn agentbase runtime update <id> --file rt.yaml
```

Update from flags:

```bash
grn agentbase runtime update <id> \
  --image-url registry.example.com/my-agent:v2 \
  --flavor-id agn-flavor-small \
  --env LOG_LEVEL=info
```

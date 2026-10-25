# runtime create

Create a new agent runtime.

## Description

Create a new agent runtime.

By default the runtime is built from flags (the simple path). For environment
variables, private-registry auth, network placement (VPC), inbound auth (IAM or
JWT), a CIDR allowlist, an MCP protocol runtime, or multi-element command/args,
use `--file` with a template produced by [generate](generate.md).

The runtime is created asynchronously; this command returns as soon as the
service accepts the spec (state `CREATING`). Converge with
`grn agentbase runtime wait <id>`.

When `--file` is set it is authoritative — every flag below is ignored and the
file must contain `name`, `imageUrl`, and `flavorId`.

> **Deprecation:** `--min-replicas` is gone. The platform owns the autoscaling
> floor (minimum replicas is always 1); only `--max-replicas` is configurable.

## Synopsis

```text
grn agentbase runtime create
    --name <value>
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
    [--poc]
    [--file <spec.yaml>]
```

## Options

**`--name`** (string)

Runtime name (immutable). Required unless `--file` is set.

- Required: Yes
- Default: (empty)
- Alias: `-n`
- Constraints: 5–50 chars; sealed (immutable) after creation.

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

Inbound CIDR allowlist (repeatable, max 20). Applies to the runtime's inbound
traffic on `PUBLIC` networks.

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

**`--poc`** (bool)

Mark as a proof-of-concept runtime. Create-only: dropped on update.

- Required: No
- Default: `false`

**`--file`** (string)

Apply a spec file (see [generate](generate.md)); authoritative when set.

- Required: No
- Default: (empty)
- Constraints: when set, the flags above are ignored and the file must contain `name`, `imageUrl`, and `flavorId`.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Create a runtime from a spec file (recommended):

```bash
grn agentbase runtime generate > rt.yaml
# ...edit rt.yaml...
grn agentbase runtime create --file rt.yaml
grn agentbase runtime wait <id>
```

Create a runtime from flags:

```bash
grn agentbase runtime create \
  --name my-runtime \
  --image-url registry.example.com/my-agent:latest \
  --flavor-id agn-flavor-small \
  --command python --command -m --command my_agent \
  --env LOG_LEVEL=info
```

Create an MCP-protocol runtime in a VPC with a CIDR allowlist and JWT inbound
auth (prefer `--file` for a JWKS payload):

```bash
grn agentbase runtime create \
  --name my-mcp-runtime \
  --image-url registry.example.com/my-mcp:latest \
  --flavor-id agn-flavor-small \
  --protocol MCP \
  --network-mode VPC --vpc-id vpc-123 --subnet-id subnet-456 \
  --allowed-cidrs 10.10.0.0/16 \
  --inbound-mode JWT --inbound-jwt-source DISCOVERY \
  --inbound-jwt-discovery-url https://idp.example.com/.well-known/openid-configuration
```

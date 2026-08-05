# gateway create

Create a new MCP gateway.

## Description

Create a new MCP gateway.

By default the gateway is built from flags (the simple path: no inline targets).
For targets, outbound auth, JWT inbound auth, or anything nested, use `--file`
with a template produced by `grn agentbase gateway generate`.

The gateway is created asynchronously; this command returns as soon as the
service accepts the spec (state `WAITING_CREATING`). Converge with
`grn agentbase gateway wait <name>`.

When `--file` is set it is authoritative and the individual flags below are
ignored. `--file` (produced by [generate](generate.md)) is the recommended path
for complex nested configuration (targets / outboundAuth / inboundAuth /
privateNetwork).

## Synopsis

```text
grn agentbase gateway create
    --name <value>
    --network-mode <value>
    --flavor-id <value>
    --replicas <value>
    --inbound-mode <value>
    [--display-name <value>]
    [--description <value>]
    [--policy-group-id <value>]
    [--client-redirect-uri <value>]
    [--iam-require-owner]
    [--jwt-source <value>]
    [--jwt-discovery-url <value>]
    [--jwt-jwks <value>]
    [--allowed-audience <value>]
    [--allowed-client <value>]
    [--allowed-scope <value>]
    [--principal-claim <value>]
    [--private-vpc-id <value>]
    [--private-subnet-id <value>]
    [--private-route <value>]
    [--public-endpoint-enabled]
    [--allowed-cidr <value>]
    [--host-alias <value>]
    [--file <value>]
```

## Options

**`--name`** (string)

Gateway name.

- Required: Yes (or supply `--interactive` to be prompted)
- Alias: `-n`
- Constraints: 3–40 characters, `[a-z0-9-]`; sealed at create.

**`--display-name`** (string)

Display name.

- Required: No
- Default: *(none)*

**`--description`** (string)

Description.

- Required: No
- Default: *(none)*

**`--network-mode`** (string)

Network mode. Sealed at create (recreate the gateway to change).

- Required: Yes (or `--interactive`)
- Possible values: `PUBLIC`, `PRIVATE`

**`--flavor-id`** (string)

Flavor id (a catalog flavor). Sealed at create.

- Required: Yes (or `--interactive`)

**`--replicas`** (integer)

Replica count. Sealed at create.

- Required: Yes (or `--interactive`)
- Constraints: `1`–`10`

**`--inbound-mode`** (string)

Inbound (caller) authentication mode.

- Required: Yes (or `--interactive`)
- Possible values: `NONE`, `IAM`, `JWT`

**`--client-redirect-uri`** (list&lt;string&gt;)

Allowed client redirect URI for DCR/authorize. *(repeatable)*

- Required: No
- Syntax: repeat the flag, e.g. `--client-redirect-uri https://a/cb --client-redirect-uri https://b/cb`

**`--iam-require-owner`** (boolean)

IAM mode only: require the caller to own the resource. Only sent when explicitly
set.

- Required: No
- Default: `false`

**`--jwt-source`** (string)

JWT inbound-auth source. Triggers JWT config assembly when set (or when
`--inbound-mode` is `JWT`).

- Required: No
- Possible values: `DISCOVERY`, `JWKS`

**`--jwt-discovery-url`** (string)

JWT OIDC discovery URL (used when `--jwt-source` is `DISCOVERY`).

- Required: No

**`--jwt-jwks`** (string)

JWT inline JWKS document (used when `--jwt-source` is `JWKS`).

- Required: No

**`--allowed-audience`** (list&lt;string&gt;)

Allowed JWT audience. *(repeatable)*

- Required: No

**`--allowed-client`** (list&lt;string&gt;)

Allowed JWT client id. *(repeatable)*

- Required: No

**`--allowed-scope`** (list&lt;string&gt;)

Allowed JWT scope. *(repeatable)*

- Required: No

**`--principal-claim`** (string)

JWT principal claim.

- Required: No
- Default: `sub`

**`--policy-group-id`** (string)

Policy group id to bind (foreign key).

- Required: No

**`--private-vpc-id`** (string)

PRIVATE-mode VPC id. Sealed at create. Required when `--network-mode` is
`PRIVATE`; forbidden otherwise (the command errors if these flags are set in
PUBLIC mode).

- Required: Conditional — required when `--network-mode` is `PRIVATE`.

**`--private-subnet-id`** (string)

PRIVATE-mode subnet id. Sealed at create.

- Required: Conditional — required when `--network-mode` is `PRIVATE`.

**`--private-route`** (list&lt;string&gt;)

PRIVATE mode: node route CIDRs the worker programs as node routes. *(repeatable)*

- Required: No

**`--public-endpoint-enabled`** (boolean)

PRIVATE mode: also expose a public endpoint. Only sent when explicitly set.

- Required: No
- Default: `false`

**`--allowed-cidr`** (list&lt;string&gt;)

Inbound client-IP allowlist as IPv4 CIDRs. Omit to allow all (`0.0.0.0/0`); an
explicit empty list blocks all client IPs. *(repeatable)*

- Required: No

**`--host-alias`** (list&lt;string&gt;)

`/etc/hosts` overrides applied to gateway pods, in `ip=host1,host2` form.
*(repeatable)*

- Required: No
- Syntax: `--host-alias 10.0.0.1=foo.local,bar.local`

**`--file`** (string)

Apply a spec file (YAML or JSON; see [generate](generate.md)). Authoritative
when set — individual flags above are ignored.

- Required: No

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters instead of
  erroring
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Create a simple PUBLIC gateway from flags:

```bash
grn agentbase gateway create \
  --name my-gateway \
  --network-mode PUBLIC \
  --flavor-id fill-flavor-id \
  --replicas 1 \
  --inbound-mode NONE
grn agentbase gateway wait my-gateway
```

Create from a spec file (recommended for nested config):

```bash
grn agentbase gateway generate > gw.yaml
# ...edit gw.yaml...
grn agentbase gateway create --file gw.yaml
grn agentbase gateway wait my-gateway
```

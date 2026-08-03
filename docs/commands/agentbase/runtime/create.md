# runtime create

Create a new agent runtime.

## Description

Create a new agent runtime.

By default the runtime is built from flags (the simple path). For environment
variables, private-registry auth, or multi-element command/args, use `--file`
with a template produced by [generate](generate.md).

The runtime is created asynchronously; this command returns as soon as the
service accepts the spec (state `CREATING`). Converge with
`grn agentbase runtime wait <id>`.

When `--file` is set it is authoritative — every flag below is ignored and the
file must contain `name`, `imageUrl`, and `flavorId`.

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
    [--command <value>]...
    [--args <value>]...
    [--env KEY=VALUE]...
    [--min-replicas <value>]
    [--max-replicas <value>]
    [--cpu-utilization <value>]
    [--memory-utilization <value>]
    [--file <spec.yaml>]
```

## Options

**`--name`** (string)

Runtime name (immutable). Required unless `--file` is set.

- Required: Yes
- Default: (empty)
- Alias: `-n`
- Constraints: sealed (immutable) after creation.

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

**`--min-replicas`** (int)

Autoscaling: minimum replicas (1-10).

- Required: No
- Default: `1`
- Constraints: 1–10; must be ≤ `--max-replicas`.

**`--max-replicas`** (int)

Autoscaling: maximum replicas (1-10).

- Required: No
- Default: `2`
- Constraints: 1–10; must be ≥ `--min-replicas`.

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

**`--file`** (string)

Apply a spec file (see [generate](generate.md)); authoritative when set.

- Required: No
- Default: (empty)
- Constraints: when set, the flags above are ignored and the file must contain `name`, `imageUrl`, and `flavorId`.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

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

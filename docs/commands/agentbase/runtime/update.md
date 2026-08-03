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

## Synopsis

```text
grn agentbase runtime update <id>
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

Apply a full-spec file (create template minus `name`).

- Required: No
- Default: (empty)
- Constraints: when set, the flags above are ignored. The file is validated as a full create spec, so a non-empty `name` must be present (its value is ignored; the runtime is addressed by the `<id>` argument).

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Update a runtime from an edited spec file, then converge:

```bash
grn agentbase runtime generate > rt.yaml
# ...edit rt.yaml (keep name set)...
grn agentbase runtime update agn-rt-abc123 --file rt.yaml
grn agentbase runtime wait agn-rt-abc123
```

Update a runtime's image and autoscaling from flags:

```bash
grn agentbase runtime update agn-rt-abc123 \
  --image-url registry.example.com/my-agent:v2 \
  --flavor-id agn-flavor-small \
  --max-replicas 4
```

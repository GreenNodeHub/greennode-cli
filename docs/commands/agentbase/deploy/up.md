# deploy up

Apply an agent (create-if-absent across services) and converge.

## Description

Apply an agent manifest: create the identity, optional memory, and runtime if absent (each looked up by name), then converge the runtime to `ACTIVE`. Existing resources are left as-is (memory has no update; runtime is not re-applied) — re-run is safe and idempotent.

`imageAuth: auto` in the manifest resolves the runtime's private-registry pull credentials from your vCR robot account — `up` calls `cr registry-credential get` and injects the resolved `{username, secret}` as the runtime's image-pull credentials (this is how `cr` is wired into `deploy`).

Use `--no-wait` to return as soon as the runtime is submitted (state `CREATING`) without converging.

`up` is **fire-and-report**: if a later step fails, earlier resources are not rolled back. The error message names what was already applied; re-run `up` (idempotent) to retry, or `destroy <name>` to tear down.

## Synopsis

```text
grn agentbase deploy up
    --file <path>
    [--name <value>]
    [--description <value>]
    [--image <value>]
    [--flavor-id <value>]
    [--image-auth <value>]
    [--command <value>]
    [--args <value>]
    [--env <value>]
    [--min-replicas <value>]
    [--max-replicas <value>]
    [--cpu-utilization <value>]
    [--memory-utilization <value>]
    [--memory-strategy <value>]
    [--set-current]
    [--no-wait]
    [--timeout <value>]
    [--interval <value>]
```

With `--file`, only the file is authoritative; the flags below build a minimal manifest for the simple (no-file) path.

## Options

**`--file`** (string)

Apply a manifest file (see [deploy generate](generate.md)).

- Required: No
- When provided, the file is authoritative and the other flags are ignored.

**`--name`** (string)

Agent name (the shared join key; required without `--file`).

- Required: Conditional — required without `--file`.
- Constraints: 3–50 characters, `^[a-zA-Z0-9_-]+$`.

**`--description`** (string)

Description.

- Required: No

**`--image`** (string)

Container image URL (required without `--file`).

- Required: Conditional — required without `--file`.

**`--flavor-id`** (string)

Flavor id (required without `--file`).

- Required: Conditional — required without `--file`.

**`--image-auth`** (string)

Private-registry auth: `auto` (resolve from `cr`); explicit via `--file` only.

- Required: No
- Possible values: `auto`
- Constraints: must be `auto`; explicit `{username, password}` is only accepted inside a `--file` manifest.

**`--command`** (list&lt;string&gt;)

Container entrypoint element (repeatable).

- Required: No
- Syntax: `--command ./agent`

**`--args`** (list&lt;string&gt;)

Container arg (repeatable).

- Required: No
- Syntax: `--args --port --args 8080`

**`--env`** (list&lt;string&gt;)

Environment variable `KEY=VALUE` (repeatable).

- Required: No
- Syntax: `--env LOG_LEVEL=info`

**`--min-replicas`** (integer)

Autoscaling min replicas.

- Required: No
- Default: `1`

**`--max-replicas`** (integer)

Autoscaling max replicas.

- Required: No
- Default: `2`
- Constraints: must be greater than or equal to `--min-replicas`.

**`--cpu-utilization`** (integer)

Autoscaling target CPU utilization (10–90).

- Required: No
- Default: `70`

**`--memory-utilization`** (integer)

Autoscaling target memory utilization (10–90).

- Required: No
- Default: `70`

**`--memory-strategy`** (list&lt;string&gt;)

Memory strategy `TYPE` (repeatable; e.g. `USER_PREFERENCE`). Adds a memory container.

- Required: No
- Syntax: `--memory-strategy USER_PREFERENCE`
- Constraints: omit for a stateless agent; supplying any value creates the memory container with one strategy per flag (default namespace template `/strategies/<TYPE>/actors/{actorId}`).

**`--set-current`** (boolean)

Set the created identity as the current agent in the profile.

- Required: No
- Default: `false`

**`--no-wait`** (boolean)

Return as soon as the runtime is submitted (do not converge to `ACTIVE`).

- Required: No
- Default: `false`

**`--timeout`** (duration)

Maximum time to wait for the runtime to converge.

- Required: No
- Default: `10m`

**`--interval`** (duration)

Poll interval while converging.

- Required: No
- Default: `5s`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Apply a manifest file (recommended path):

```bash
grn agentbase deploy up --file agent.yaml
```

Simple path from flags, resolving image pull creds from `cr`, and setting the identity as current:

```bash
grn agentbase deploy up \
  --name my-agent \
  --image registry.vngcloud.vn/myrepo/my-agent:v1 \
  --flavor-id agent.small \
  --image-auth auto \
  --set-current
```

Submit the runtime without waiting for it to converge:

```bash
grn agentbase deploy up --file agent.yaml --no-wait
```

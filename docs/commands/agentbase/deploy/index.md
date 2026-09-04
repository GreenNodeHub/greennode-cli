# deploy

Deploy and manage an agent as one unit. `deploy` is a **client-side orchestrator** — it has no backend of its own. It composes the `access`, `memory`, `runtime`, and `cr` clients into a single lifecycle.

An **agent** is the set of resources that share a name (the join key): an identity (always), an optional memory container (stateless agents omit it), and a runtime (the container that runs the agent code). There is no cross-service foreign key — the name is what ties them together. The agent code is a container image; push it to your vCR repo and reference it in the manifest. `imageAuth: auto` resolves the pull credentials from your auto-provisioned robot account.

```bash
grn agentbase deploy <command> [options]
```

## Available commands

| Command | Description |
|---------|-------------|
| [generate](generate.md) | Print an agent manifest template (YAML or JSON) |
| [up](up.md) | Apply an agent (create-if-absent across services) and converge |
| [status](status.md) | Show the cross-service state of an agent |
| [destroy](destroy.md) | Delete an agent's runtime and memory (and identity with `--purge`) |

`up` is idempotent (create-if-absent per service, then converge the runtime to `ACTIVE` unless `--no-wait`). `destroy` deletes runtime + memory; pass `--purge` to also delete the identity. `up`/`status`/`destroy` look resources up by name, so no state file is needed. On failure, `up` is **fire-and-report** — it does not roll back already-applied resources; re-run `up` (idempotent) or `destroy` to retry.

## Manifest

`up` takes a manifest file (`--file`) describing the agent, or individual flags for the simple path. Generate a commented template with [deploy generate](generate.md):

```bash
grn agentbase deploy generate > agent.yaml
```

Manifest shape (save it, fill it in, apply with `grn agentbase deploy up --file agent.yaml`):

```yaml
# name is the shared join key across identity + memory + runtime (3-50 chars,
# ^[a-zA-Z0-9_-]+$). identity is always created. memory is OPTIONAL — delete the
# whole block for a stateless agent. runtime runs the agent code as a container.
name: my-agent
description: "A customer-support agent"

identity:
  allowedReturnUrls:
    - https://app.example.com/callback

# memory: OPTIONAL. Omit the whole block for a stateless agent. When present,
# at least one strategy (name/type/namespaceTemplate) is required.
memory:
  eventExpiryDuration: 3600
  strategies:
    - name: prefs
      type: USER_PREFERENCE                      # built-in key (USER_PREFERENCE|SEMANTIC|CUSTOM|...)
      namespaceTemplate: "/strategies/USER_PREFERENCE/actors/{actorId}"

runtime:
  image: registry.vngcloud.vn/<your-repo>/my-agent:v1
  imageAuth: auto                                # "auto" resolves pull creds from your vCR robot
                                                 # account; or {username: ..., password: ...}
  command: [./agent]
  args: [--port, "8080"]
  env: {LOG_LEVEL: info}
  flavorId: agent.small
  autoscaling: {minReplicas: 1, maxReplicas: 3, cpuUtilization: 70, memoryUtilization: 80}
```

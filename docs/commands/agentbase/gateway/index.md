# gateway

Manage MCP gateways (the agent-core-gateway service).

```bash
grn agentbase gateway <command> [options]
```

Gateways are **asynchronous** resources. `create`, `update`, and `delete`
return immediately with the resource in a transitional state and converge in
the background. The lifecycle FSM is `WAITING_CREATING → CREATING → ACTIVE`
(and `WAITING_UPDATING → UPDATING → ACTIVE`, `WAITING_DELETING → DELETING →
DELETED`); failures land in a terminal `*_ERROR` state (`CREATE_ERROR`,
`UPDATE_ERROR`, `ERROR`, `ERROR_DELETING`). Use [wait](wait.md) to block until a
terminal state.

## Available commands

| Command | Description |
|---------|-------------|
| [create](create.md) | Create a new MCP gateway |
| [generate](generate.md) | Print a gateway create template (YAML or JSON) |
| [list](list.md) | List gateways |
| [get](get.md) | Show a gateway |
| [update](update.md) | Update a gateway's mutable fields |
| [delete](delete.md) | Delete a gateway |
| [wait](wait.md) | Wait for a gateway to reach a terminal state |

## Recommended workflow for complex specs

Gateway creation exposes many nested, mutually-exclusive fields (targets,
outboundAuth, inboundAuth/JWT, privateNetwork) that cannot all be expressed as
flags. For anything beyond the simple path, generate a template, fill it in, and
apply it with `--file`:

```bash
grn agentbase gateway generate > gw.yaml
# ...edit gw.yaml...
grn agentbase gateway create --file gw.yaml
grn agentbase gateway wait my-gateway
```

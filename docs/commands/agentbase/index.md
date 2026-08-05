# `grn agentbase` Reference

`grn agentbase` is the subcommand tree of the `greennode-cli` for the
**GreenNode AgentBase** platform — the set of backend services that let you
create and operate agents on VNG Cloud. It ships in the default `grn` binary
and the public release build — no special build tag is required.

```bash
grn agentbase --help
```

## Overview

The AgentBase platform consists of **6 backend services** behind
`agentbase.api.vngcloud.vn` (prod) / `agentbase.api-dev.vngcloud.tech` (dev),
plus a client-side orchestrator, `deploy`:

| Command group | Backend service | Role |
|---|---|---|
| `identity` | agent-core-identity | Manage agent digital identities + outbound auth (OAuth2/API key) |
| `gateway` | agent-core-gateway | Create/manage MCP gateways (provision VKS runtime, DNS, load balancer, quota) |
| `runtime` | agent-core-runtime | Deploy the container that runs the agent code (image/command/args/env/autoscaling) |
| `catalog` | agent-core-runtime | Browse the catalog: runtime flavors, OpenClaw versions + workspaces |
| `memory` | agent-core-memory | Long-term memory for agents (Mem0 vector store, semantic search) |
| `policy` | agent-core-policy | Cedar policy engine (policy-groups + policies + decision) |
| `cr` | agent-core-container-registry | Wrapper for VNG Cloud Container Registry (vCR): auto-provisioned repo + robot account |
| `deploy` | *(no backend)* | Orchestrator composing identity + memory + runtime (+ cr) |
| `context` | — | Show the active environment + endpoints |

An **agent** is the set of resources that share a **name** (the join key): an
identity (always present), an optional memory container (omit for a stateless
agent), and a runtime (the container that runs the agent code). There is no
cross-service foreign key; `deploy` uses the name as the join.

---

## Setup & authentication

`agentbase` **shares the `~/.greennode` profile** with the rest of `grn`
(vks/vserver) — there is no separate `.greennode.json` file. There are two auth
modes:

### Machine mode (M2M) — recommended for CI

```bash
grn configure --profile default
# Enter client_id (GRN_ACCESS_KEY_ID) and client_secret (GRN_SECRET_ACCESS_KEY)
```

The CLI uses `clientcredentials` to mint an IAM v2 token. The `client_secret` is
never baked into the binary or logged; the `refresh_token` is stored at rest
(0600, masked in `configure list/get`).

### User mode (PKCE) — recommended for interactive use

```bash
grn login --iam-env prod          # PKCE login
grn logout                        # clear the login
```

The active mode is determined by `auth_mode` in the profile (`user` or
`machine`). All three services (vks/vserver/agentbase) use a single shared token
selector.

---

## Choosing the environment (dev / prod)

The environment is selected via `iam_env` in the profile (default `prod`).

```bash
# Shared with vks/vserver:
grn configure set iam_env <dev|prod>     # (machine)
grn login --iam-env <dev|prod>           # (user)

# Show the active environment + endpoints:
grn agentbase context current
```

> Note: in **user** mode, `iam_env` is bound to the login token. To switch
> environments, you must `grn login --iam-env <env>` again. In machine mode,
> `iam_env` can be switched freely.

---

## Command reference

All commands accept the common flags `-o json|table|id` (output format) and
`--interactive` (prompt for missing required parameters).

| Command group | Description |
|---|---|
| [context](context/index.md) | Show the active environment + standard headers/decorators |
| [identity](identity/index.md) | Workload identities + outbound auth (OAuth2 / static API key / delegated) |
| [gateway](gateway/index.md) | Create/manage MCP gateways (async FSM; use `wait` to converge) |
| [runtime](runtime/index.md) | Deploy the agent container (image/command/args/env/autoscaling) |
| [memory](memory/index.md) | Long-term memory container (Mem0 vector store, semantic search) |
| [catalog](catalog/index.md) | Runtime placement flavors + OpenClaw versions/workspaces |
| [policy](policy/index.md) | Cedar policy engine (policy-groups + policies + decision) |
| [cr](cr/index.md) | VNG Cloud Container Registry wrapper (auto-provisioned repo + robot account) |
| [deploy](deploy/index.md) | Client-side orchestrator (identity + memory + runtime + cr) |

---

## Output format

Every command takes `-o` (`--output`):

| Value | Meaning |
|---|---|
| `table` (default) | Human-readable table; secrets masked |
| `json` | Raw JSON — secrets revealed (e.g. to pipe into `docker login`) |
| `id` | Print only the ID (for scripting) |

---

## Paging envelope reference

Each service uses a different pagination shape — a common source of confusion:

| Service | Envelope | `size` query key |
|---|---|---|
| gateway | `{items, pagination}` | — |
| runtime | `{listData, page, pageSize, totalPage, totalItem}` | `size` |
| memory | `{listData, page, pageSize, totalPage, totalItem}` | `size` |
| policy | `{content, page, pageSize, totalPage, totalItem}` | `page_size` (snake) |
| cr | `{data, page, pageSize, totalItem, totalPage}` | `size` |
| identity | `{content, page, size, totalElements, totalPages, ...}` | `size` |

---

## Troubleshooting

- **`authentication failed: ...`** — run `grn configure` (machine) or
  `grn login` (user). Check `grn agentbase context current` for env/profile.
- **404 on lookup** — `deploy`/`status` look up by name via `List` + filter; a
  wrong name or a resource in a different environment reports as absent.
- **`runtime` stuck in CREATING** — runtimes converge asynchronously; use
  `grn agentbase runtime wait <id>` (or `deploy status <name>`) to wait.
- **Switching dev↔prod** — machine: `grn configure set iam_env <env>`; user:
  `grn login --iam-env <env>` (must re-login, since the token is bound to env).

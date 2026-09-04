# access

Manage authentication and agent identities.

```bash
grn agentbase access <command> [options]
```

## Available commands

### Agent ID

| Command | Description |
|---------|-------------|
| [agent-id create](agent-id-create.md) | Create a new agent identity |
| [agent-id list](agent-id-list.md) | List agent identities |
| [agent-id get](agent-id-get.md) | Get an agent identity by name |
| [agent-id update](agent-id-update.md) | Update an agent identity |
| [agent-id use](agent-id-use.md) | Set the current agent identity |
| [agent-id delete](agent-id-delete.md) | Delete an agent identity |

### Outbound Auth — Static / API key

| Command | Description |
|---------|-------------|
| [outbound-auth static create](outbound-auth-static-create.md) | Create a static API key provider |
| [outbound-auth static list](outbound-auth-static-list.md) | List static API key providers |
| [outbound-auth static get](outbound-auth-static-get.md) | Get a static API key provider |
| [outbound-auth static update](outbound-auth-static-update.md) | Update a static API key provider |
| [outbound-auth static delete](outbound-auth-static-delete.md) | Delete a static API key provider |
| [outbound-auth static get-key](outbound-auth-static-get-key.md) | Get the API key for an agent identity |

### Outbound Auth — Delegated

| Command | Description |
|---------|-------------|
| [outbound-auth delegated create](outbound-auth-delegated-create.md) | Create a delegated API key provider |
| [outbound-auth delegated list](outbound-auth-delegated-list.md) | List delegated API key providers |
| [outbound-auth delegated get](outbound-auth-delegated-get.md) | Get a delegated API key provider |
| [outbound-auth delegated delete](outbound-auth-delegated-delete.md) | Delete a delegated API key provider |
| [outbound-auth delegated get-key](outbound-auth-delegated-get-key.md) | Obtain a delegated API key for an agent identity |

### Outbound Auth — OAuth2

| Command | Description |
|---------|-------------|
| [outbound-auth oauth2 create](outbound-auth-oauth2-create.md) | Create an OAuth2 provider |
| [outbound-auth oauth2 list](outbound-auth-oauth2-list.md) | List OAuth2 providers |
| [outbound-auth oauth2 get](outbound-auth-oauth2-get.md) | Get an OAuth2 provider |
| [outbound-auth oauth2 update](outbound-auth-oauth2-update.md) | Update an OAuth2 provider |
| [outbound-auth oauth2 delete](outbound-auth-oauth2-delete.md) | Delete an OAuth2 provider |
| [outbound-auth oauth2 m2m-token](outbound-auth-oauth2-m2m-token.md) | Get an M2M OAuth2 token |
| [outbound-auth oauth2 3lo-token](outbound-auth-oauth2-3lo-token.md) | Get a 3-legged OAuth2 token |

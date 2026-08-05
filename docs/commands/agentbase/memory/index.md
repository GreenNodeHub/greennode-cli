# memory

Manage agent memories — containers for an agent's long-term facts (memory records, backed by an external Mem0 vector store) and short-term conversation events.

Memories are created and deleted **synchronously** (there is no async state machine, so there is no `wait` command). Deletion is a soft delete: a memory transitions `ACTIVE → DELETED` rather than being permanently removed.

```bash
grn agentbase memory <command> [options]
```

## Available commands

| Command | Description |
|---------|-------------|
| [create](create.md) | Create a new agent memory |
| [generate](generate.md) | Print a memory create template (YAML or JSON) |
| [list](list.md) | List memories |
| [get](get.md) | Show a memory |
| [delete](delete.md) | Delete a memory |
| [search](search.md) | Semantic-search a memory's long-term facts |

## Sub-resource groups

| Group | Commands |
|-------|----------|
| [actor](actor-list.md) | [`actor list <id>`](actor-list.md) |
| [session](session-list.md) | [`session list <id> <actor-id>`](session-list.md) |
| [event](event-list.md) | [`event list`](event-list.md) · [`event create`](event-create.md) · [`event delete`](event-delete.md) |
| [strategy](strategy-list.md) | [`strategy list <id>`](strategy-list.md) |
| [record](record-list.md) | [`record list`](record-list.md) · [`record delete`](record-delete.md) · [`record insert`](record-insert.md) · [`record generate-from-session`](record-generate-from-session.md) · [`record generate-from-content`](record-generate-from-content.md) |

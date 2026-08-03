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

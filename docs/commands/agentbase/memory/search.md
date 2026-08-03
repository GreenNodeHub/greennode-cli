# memory search

Semantic-search a memory's long-term facts.

## Description

Run a semantic search over a memory's long-term memory records (backed by the external Mem0 vector store). Returns ranked facts with relevance scores.

`namespace` is the resolved namespace string the records live under (from the memory's strategy `namespaceTemplate`, e.g. `/strategies/SEMANTIC/actors/<actor>`).

In table format each result row shows score, memory text, and updated time; the resolved namespace, query, and result count are printed on stderr.

## Synopsis

```text
grn agentbase memory search <id>
    --namespace <value>
    --query <value>
    [--limit <number>]
    [--threshold <value>]
```

Positional argument:

- `<id>` — the memory ID. Exactly one argument is required.

## Options

**`--namespace`** (string)

Resolved namespace (required, e.g. `/strategies/SEMANTIC/actors/<actor>`).

- Required: Yes (enforced in the command body — errors when empty; **not** prompted by `--interactive`)

**`--query`** (string)

Search query (required).

- Required: Yes (enforced in the command body — errors when empty; **not** prompted by `--interactive`)

**`--limit`** (integer)

Max results (5-200).

- Required: No
- Default: `100`
- Constraints: 5–200.

**`--threshold`** (float)

Min relevance score (0-1).

- Required: No
- Default: `0`
- Constraints: 0–1.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Semantic search for a fact under a resolved namespace:

```bash
grn agentbase memory search mem-123 \
  --namespace /strategies/SEMANTIC/actors/alice \
  --query "dark mode"
```

Limit results and require a minimum relevance score:

```bash
grn agentbase memory search mem-123 \
  --namespace /strategies/SEMANTIC/actors/alice \
  --query "dark mode" \
  --limit 20 \
  --threshold 0.5
```

# memory create

Create a new agent memory.

## Description

Create a new agent memory.

Creation requires at least one long-term-memory strategy (each with name, type, and a namespace template). The simple flag path builds a single strategy; for multiple strategies or a custom fact-extraction prompt, use `--file` with a template produced by `grn agentbase memory generate`.

The memory is created synchronously and is immediately `ACTIVE` (no wait needed).

When `--file` is set it is authoritative — the spec file is parsed and the flags below are ignored. Otherwise the flags build a single `longTermMemoryStrategies` entry.

## Synopsis

```text
grn agentbase memory create
    -n, --name <value>
    --description <value>
    [--event-expiry-duration <days>]
    --strategy-name <value>
    --strategy-type <value>
    --strategy-namespace <value>
    [--strategy-prompt <value>]
    [--strategy-auto-generate]
    [--file <value>]
```

## Options

**`--name`** (`-n`) (string)

Memory name (required without `--interactive`).

- Required: Yes (enforced on the flag path unless `--file` is set; prompted when `--interactive`)
- Constraints: matches `^[a-zA-Z0-9._-]*$`, max 50 characters.

**`--description`** (string)

Description (required).

- Required: Yes (enforced on the flag path unless `--file` is set; prompted when `--interactive`)

**`--event-expiry-duration`** (integer)

Short-term event TTL in days (1-365).

- Required: No
- Default: `30`
- Constraints: 1–365.

**`--strategy-name`** (string)

Single strategy: name (use `--file` for multiple).

- Required: Yes (enforced on the flag path unless `--file` is set; prompted when `--interactive`)

**`--strategy-type`** (string)

Single strategy: type (`USER_PREFERENCE|SEMANTIC|CUSTOM|...`).

- Required: Yes (enforced on the flag path unless `--file` is set; prompted when `--interactive`)
- Possible values: built-in strategy keys, e.g. `USER_PREFERENCE`, `SEMANTIC`, `CUSTOM`. Uppercased before send.

**`--strategy-namespace`** (string)

Single strategy: namespace template (e.g. `/strategies/SEMANTIC/actors/{actorId}`).

- Required: Yes (enforced on the flag path unless `--file` is set; prompted when `--interactive`)

**`--strategy-prompt`** (string)

Single strategy: custom fact-extraction prompt (max 1000).

- Required: No
- Constraints: max 1000 characters.

**`--strategy-auto-generate`** (boolean)

Single strategy: auto-generate memory records from events.

- Required: No
- Default: `false` (only sent when explicitly set)

**`--file`** (string)

Apply a spec file (see `generate`); authoritative when set.

- Required: No
- Notes: A YAML or JSON spec of a `CreateMemoryRequest`. When set, all flags above are ignored and the file's `name` and `longTermMemoryStrategies` (each with `name`/`type`/`namespaceTemplate`) must be present.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Create a memory with a single SEMANTIC strategy (flag path):

```bash
grn agentbase memory create \
  -n my-memory \
  --description "Alice's long-term facts" \
  --strategy-name semantic \
  --strategy-type SEMANTIC \
  --strategy-namespace /strategies/SEMANTIC/actors/{actorId}
```

Create from a spec file (multiple strategies / custom prompt):

```bash
grn agentbase memory generate > mem.yaml
# edit mem.yaml, then:
grn agentbase memory create --file mem.yaml
```

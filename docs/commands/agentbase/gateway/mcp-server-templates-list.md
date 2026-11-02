# gateway mcp-server-templates list

List MCP server templates from the catalog.

## Description

List enabled MCP server templates from the catalog (GET
`/api/v1/mcp-server-templates`). Each item carries auth options whose
`recommended*` fields pre-fill a target-creation form; edits go through the
gateway update path. `displayName`/`vendor`/`id` are case-insensitive
substring filters that AND together.

## Synopsis

```text
grn agentbase gateway mcp-server-templates list
    [--display-name <value>]
    [--vendor <value>]
    [--id <value>]
    [--page <value>]
    [--size <value>]
```

## Options

**`--display-name`** (string)

Filter by display name (substring, case-insensitive).

- Required: No
- Default: (empty)

**`--vendor`** (string)

Filter by vendor (substring, case-insensitive).

- Required: No
- Default: (empty)

**`--id`** (string)

Filter by template id (substring, case-insensitive).

- Required: No
- Default: (empty)

**`--page`** (int)

Page number (1-based).

- Required: No
- Default: `1`

**`--size`** (int)

Page size (max 100).

- Required: No
- Default: `50`

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase gateway mcp-server-templates list
grn agentbase gateway mcp-server-templates list --vendor github
grn agentbase gateway mcp-server-templates list --display-name slack --vendor slack
```

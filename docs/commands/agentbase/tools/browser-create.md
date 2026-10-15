# tools browser create

Create a headless browser.

## Description

Create a platform-managed headless browser (POST `/v1/browsers`).

Creation is **asynchronous**: the API responds 201 with the browser in
`CREATING`. Run [`browser wait`](browser-wait.md) until it is `ACTIVE`
before attaching sessions — creating a session on a non-ACTIVE browser fails
with 409.

## Synopsis

```text
grn agentbase tools browser create
    --name <value>
    [--description <value>]
```

## Options

**`--name`** (string) — browser name, at most 255 characters (validated
client-side). Required (or pass `-i` to prompt).

**`--description`** (string) — browser description, at most 1000
characters (validated client-side). Optional.

## Output

- `table` — the new browser (field/value)
- `json` — the raw browser object
- `id` — the new browser's id

On success a hint to run `browser wait <id>` is printed to stderr.

## Examples

```bash
grn agentbase tools browser create --name my-browser --description "docs scraper"
grn agentbase tools browser create -i          # prompt for the name
grn agentbase tools browser create --name my-browser -o id   # script the id
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

# tools code-interpreter create

Create a code interpreter.

## Description

Create a platform-managed code-execution sandbox
(POST `/v1/code-interpreters`).

Creation is **asynchronous**: the API responds 201 with the code interpreter
in `CREATING`. Run [`code-interpreter wait`](code-interpreter-wait.md) until
it is `ACTIVE` before attaching sessions — creating a session on a
non-ACTIVE code interpreter fails with 409.

## Synopsis

```text
grn agentbase tools code-interpreter create
    --name <value>
    [--description <value>]
```

## Options

**`--name`** (string) — code interpreter name. Required (or pass `-i` to
prompt).

**`--description`** (string) — code interpreter description. Optional.

## Output

- `table` — the new code interpreter (field/value)
- `json` — the raw code interpreter object
- `id` — the new code interpreter's id

On success a hint to run `code-interpreter wait <id>` is printed to stderr.

## Examples

```bash
grn agentbase tools code-interpreter create --name my-runner
grn agentbase tools code-interpreter create -i          # prompt for the name
grn agentbase tools code-interpreter create --name my-runner -o id
```

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

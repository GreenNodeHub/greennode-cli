# policy group policy generate

Print a policy create template (YAML or JSON).

## Description

Print a starter spec for [group policy create --file](group-policy-create.md).
With `--output json` the template is emitted as JSON; otherwise a commented YAML
skeleton is printed. The YAML skeleton includes a full `statement` (effect,
principal, actions, resources) and a commented `condition` example covering the
`when`/`unless` branches and the accepted operators.

Keys are the JSON (camelCase) field names so the file round-trips through
`create --file` exactly.

## Synopsis

```text
grn agentbase policy group policy generate
```

## Options

This command takes no command-specific options.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`); `json` selects the JSON template here
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

Print the YAML template:

```bash
grn agentbase policy group policy generate
```

Print the JSON template and save it:

```bash
grn agentbase policy group policy generate -o json > policy.json
```

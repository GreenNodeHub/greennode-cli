# gateway generate

Print a gateway create template (YAML or JSON).

## Description

Print a commented gateway create template to stdout. Save it, fill it in, and
apply with `grn agentbase gateway create --file <file>`.

Defaults to YAML (with comments); pass `-o json` for a JSON skeleton.

The template keys are the JSON (camelCase) field names, so the file round-trips
through `create --file` exactly. Required keys: `name`, `networkMode`,
`flavorId`, `replicas`, `inboundAuth.mode`. Sealed at create: `name`,
`networkMode`, `flavorId`, `replicas`, and (PRIVATE)
`privateNetwork.vpcId`/`subnetId`.

## Synopsis

```text
grn agentbase gateway generate
```

## Options

This command takes no command-specific options. Use the global `-o json` to emit
a JSON skeleton instead of the commented YAML template.

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`); for
  `generate`, `-o json` selects the JSON skeleton
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

## Examples

Print the commented YAML template and save it:

```bash
grn agentbase gateway generate > gw.yaml
```

Print a JSON skeleton:

```bash
grn agentbase gateway generate -o json > gw.json
```

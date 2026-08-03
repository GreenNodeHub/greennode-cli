# cr repository artifact delete

Delete a single artifact by digest.

## Description

Delete a single artifact (one digest) within an image, leaving the image's
other artifacts intact. Both `--image-name` and `--digest` are required (sent
to the API as the `?imageName=` and `?digest=` query parameters); on success
the API returns `204 No Content`.

If `-i/--interactive` is set, missing values are prompted; otherwise both flags
are required.

## Synopsis

```text
grn agentbase cr repository artifact delete
    --image-name <value>
    --digest <value>
```

## Options

**`--image-name`** (string)

Image name (required).

- Required: Yes

**`--digest`** (string)

Artifact digest (required).

- Required: Yes

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase cr repository artifact delete \
  --image-name my-agent \
  --digest sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
```

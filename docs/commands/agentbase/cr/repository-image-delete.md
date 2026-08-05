# cr repository image delete

Delete an image (all its artifacts/tags).

## Description

Delete an image and every artifact/tag it contains. The target is identified by
`--image-name` (sent to the API as the `?imageName=` query parameter); on
success the API returns `204 No Content`.

If `--image-name` is omitted and `-i/--interactive` is set, you are prompted for
it; otherwise `--image-name` is required.

## Synopsis

```text
grn agentbase cr repository image delete
    --image-name <value>
```

## Options

**`--image-name`** (string)

Image name to delete (required).

- Required: Yes

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`, `--endpoint-url`, `--debug`

## Examples

```bash
grn agentbase cr repository image delete --image-name my-agent
```

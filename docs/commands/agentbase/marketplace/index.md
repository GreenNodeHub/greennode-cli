# marketplace

Browse the GreenNode catalog served by the agent-core-runtime service.

## Description

The `marketplace` group fronts the catalog endpoints under `/v1/` (flavors,
openclaw versions, and the openclaw workspace registry). It uses the same
base URL as the runtime service. These operators are part of the agentbase
surface but were previously unwired in the CLI; they are now available under
`grn agentbase marketplace`.

## Subcommands

| Subcommand | Description |
| --- | --- |
| [`flavors list`](flavors-list.md) | List runtime placement flavors |
| [`openclaw-versions list`](openclaw-versions-list.md) | List OpenClaw versions |
| [`openclaw list`](openclaw-list.md) | List OpenClaw workspaces |
| [`openclaw create`](openclaw-create.md) | Create an OpenClaw workspace |
| [`openclaw get <id>`](openclaw-get.md) | Show an OpenClaw workspace |
| [`openclaw delete <id>`](openclaw-delete.md) | Delete an OpenClaw workspace |
| [`openclaw start <id>`](openclaw-start.md) | Start an OpenClaw workspace |
| [`openclaw stop <id>`](openclaw-stop.md) | Stop an OpenClaw workspace |
| [`openclaw update-version <id>`](openclaw-update-version.md) | Roll an OpenClaw workspace to a version |

## Global options

All `grn agentbase` commands accept:

- `-o, --output json|table|id` — output format (default `table`)
- `-i, --interactive` — prompt for missing required parameters
- The shared `grn` global options: `--profile`, `--region`, `--query`,
  `--endpoint-url`, `--debug`

# VLB live E2E results

Copy to `RESULTS.md`, fill in as you work through [`RUNBOOK.md`](RUNBOOK.md), and attach to the
release PR that removes the `vks_only` build tag.

**Do not paste secrets.** Redact bearer tokens, private keys, and the client ID / secret used for
the run. Before signing off:

```bash
grep -nE 'BEGIN PRIVATE KEY|Bearer ' tests/vlb/RESULTS.md   # must be empty
```

## Run metadata

| Field | Value |
|---|---|
| Date | |
| Operator | |
| Region | HCM-3 |
| Project ID | |
| Zone | HCM03-1C |
| `grn --version` | |
| Commit SHA | |
| Build | `CGO_ENABLED=0 go build -o /tmp/grn .` (no `vks_only`) |
| Credentials rotated after the run? | ☐ yes |

## Phase 0 — Preflight

| # | Check | Result | Notes |
|---|---|---|---|
| 0.1a | `vlb --help` lists 5 top-level noun groups | | |
| 0.1b | `-tags vks_only` binary has no `vlb` | | |
| 0.1c | `go test ./...` baseline green | | |
| 0.1d | aliases `lb` / `ca` resolve | | |
| 0.1e | bare groups print help, reject stray args | | |
| 0.2a | `configure list` masks credentials | | |
| 0.2b | `configure get client_secret` masks | | |
| 0.3a | endpoint is `vlb-gateway` | | |
| 0.3b | `--debug` does not leak the bearer token | | |
| 0.3c | `--no-verify-ssl` warns on stderr | | |
| 0.3d | `--region BOGUS` rejected | | |
| 0.4 | missing `project_id` fails before the request | | |
| 0.5 | bad IDs rejected before any HTTP call | | |

## Command coverage — all 37

`n/a` where the dimension does not apply. `blocked` with a note where the environment prevents it.

### loadbalancer (14)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `lb list` | | n/a | n/a | | | |
| `lb get` | | | n/a | | | |
| `lb create` | | | | | | |
| `lb delete` | | | | | | dry-run performs a GET |
| `lb resize` | | | | | | |
| `lb clone` | | | | | | |
| `lb configure-scaling` | | | | | | may need `--auto-scalable` |
| `lb update-bandwidth` | | | | | | **blocker:** no source for `bandwidth-pack-<uuid>` |
| `lb get-clone-metadata` | | | n/a | | | |
| `lb list-scale-history` | | | n/a | | | bare-array envelope |
| `lb list-headers` | | n/a | n/a | | | takes no flags at all |
| `lb list-by-subnet` | | | n/a | | | |
| `lb wait-active` | | | n/a | n/a | n/a | exit 0 / 255 |
| `lb wait-deleted` | | | n/a | n/a | n/a | exit 0 / 255 |

### listener (5)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `listener list` | | | n/a | | | shows `defaultPoolId` |
| `listener get` | | | n/a | | | |
| `listener create` | | | | | | TCP in Phase 2, HTTPS in Phase 3 |
| `listener update` | | | | | | P0: partial-update + TLS-skip |
| `listener delete` | | | | | | |

### listener policy (5)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `policy list` | | | n/a | | | positions start at 1 |
| `policy get` | | | n/a | | | |
| `policy create` | | | | | | P0: always sends `rules: []` |
| `policy update` | | | | | | rules replaced, not appended |
| `policy delete` | | | | | | |

### pool (6)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `pool list` | | | n/a | | | |
| `pool get` | | | n/a | | | omits `healthMonitor` + `members` |
| `pool create` | | | | | | health-monitor validation |
| `pool update` | | | | | | P0: two GETs, non-empty monitor |
| `pool delete` | | | | | | |
| `pool get-health-monitor` | | | n/a | | | only source of the monitor |

### pool member (2)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `member list` | | | n/a | | | |
| `member update` | | | | | | P0: replace / keep-existing / empty |

### certificate (4)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `certificate list` | | n/a | n/a | | | no pagination flags |
| `certificate get` | | | n/a | | | |
| `certificate create` | | | | | | dry-run must redact |
| `certificate delete` | | | | | | warns when `inUse` |

### package (1)

| Command | Live happy path | Required-flag error | `--dry-run` | json / table / text | `--query` | Notes |
|---|---|---|---|---|---|---|
| `package list` | | n/a | n/a | | | `--all` ignores `--zone-id` |

## Undocumented `POST` response keys

The spec declares these responses only as `additionalProperties: string`, so the key name is
whatever the service sends. Record what you actually saw — this is the reference for anyone
scripting against the CLI, and worth pinning in `go/cmd/vlb/CLAUDE.md`.

| Create command | Response key | Example value |
|---|---|---|
| `lb create` | | |
| `lb clone` | | |
| `pool create` | | |
| `listener create` | | |
| `policy create` | | |
| `certificate create` | `data.uuid` (full `CADto`, not a string map) | |

## P0 regressions

These are where real bugs have occurred. All six must be green.

| # | Regression | Result | Evidence |
|---|---|---|---|
| P0-1 | `pool update` issues two GETs; `healthMonitor` in the body is fully populated | | |
| P0-2 | `listener update` on a **TCP** listener sends no TLS fields | | |
| P0-3 | `policy create` with no `--rule` still sends `rules: []` | | |
| P0-4 | `listener update` of one timeout leaves the other two at 50 | | |
| P0-5 | `member update`: replace removes, `--keep-existing` merges, `--member ''` empties | | |
| P0-6 | ~~`policy update-order` position=0~~ — **command removed**, hazard eliminated | n/a | update_order.go deleted |

## Validation sweep (Phase 4)

| # | Group | Result | Notes |
|---|---|---|---|
| 4.1 | required flags on all 37 commands | | |
| 4.2 | health-monitor hidden required fields (a–d) | | |
| 4.2⚠ | `pool update` skips `validateHealthMonitor` — what does the API do? | | finding if 500 |
| 4.3 | member validation (a–d) | | |
| 4.4 | enums and ranges (a–j) | | |
| 4.4j | `--scheme` / `--type` not validated locally — add a check? | | decision |
| 4.5 | "nothing to update" lists every flag | | |
| 4.6 | certificate mutually-exclusive flags | | |
| 4.7 | struct-flag parsing (a–f) | | |
| 4.8 | 404 handling | | |
| 2.11 | second mutation without `wait-active` → `is not ready` | | |

## Waiters and completion (Phase 5)

| # | Check | Result | Notes |
|---|---|---|---|
| 5.1a | `wait-active` success, exit 0, stderr/stdout split | | |
| 5.1b | timeout → exit 255 | | |
| 5.1c | 404 is fatal → exit 255 | | |
| 5.1d | `wait-deleted` on `ACTIVE` → exit 255 | | |
| 5.1e | `wait-deleted` after delete → exit 0 | | |
| 5.1f | state read from `data.displayStatus` | | |
| 5.2a–j | completion matrix | | |
| 5.2 | dead `--vpc-ids` mapping — remove? | | recommendation |

## Findings

Anything that is a real bug must be fixed, covered by a test, and documented before shipping.

| # | Command | Symptom | Severity | Fixed? | Test added | Doc updated |
|---|---|---|---|---|---|---|
| 1 | | | | | | |

## Teardown

| Check | Result |
|---|---|
| `lb list --name $PREFIX` empty | |
| `certificate list --name $PREFIX` empty | |
| local `/tmp/e2e.{key,crt}` removed | |
| credentials unset and rotated | |
| `grep -nE 'BEGIN PRIVATE KEY\|Bearer ' RESULTS.md` empty | |
| `go test ./...` still green | |

## Sign-off

| Field | Value |
|---|---|
| Ready to remove `vks_only`? | ☐ yes ☐ no |
| Blockers remaining | |
| Signed off by | |

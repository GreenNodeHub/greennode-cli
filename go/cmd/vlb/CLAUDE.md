# VLB CLI — product notes

Tier-2 notes for the `grn vlb` command group. The root `CLAUDE.md` has the
repo-wide conventions; this file records what is specific to VLB. Source of
truth for the API is `openapi-vlb.json` at the repo root.

## Client & endpoint

- Build the client with **`vlbclient.Resolve(cmd)`**, which returns
  `(client, cfg, projectID, error)` in one step — every VLB path embeds the
  project ID, so resolving both together removes a repeated error check.
  `vlbclient.BuildClient` / `ProjectID` are available separately for completers.
- Endpoint: `internal/config` REGIONS key `vlb_endpoint`
  (`https://<region>.api.vngcloud.vn/vserver/vlb-gateway`) — its own gateway,
  distinct from `vserver_endpoint`.
- **`project_id` is required** and goes in the URL path (from `grn configure`
  or `GRN_DEFAULT_PROJECT_ID`).
- **Never build a path with `fmt.Sprintf` at a call site.** All URL construction
  lives in `internal/vlbclient/paths.go`, so every ID interpolation is in one
  auditable place. Call `validator.ValidateID` on an ID before passing it there.

## API quirks (differ from vks — do not copy vks assumptions)

- **Pagination is 1-based**: params `page` (from **1**) + **`size`** (note:
  `size`, not `pageSize`). Same as vServer, unlike VKS.
- Do **not** use `client.GetAllPages` — it is hardwired to VKS's 0-based
  `page`/`pageSize` and `{"items": [...]}` envelope.
- Paths are all under **`/v2/{projectID}/...`**.
- **Query params are flat, and almost none of the documented filters work.**
  Verified against the live HCM-3 API (2026-09-08):
  - `name` (substring) and `zoneId` (exact) are honoured. Flat is confirmed
    correct: `name=vks-vlb` narrows the result, `filter.name=vks-vlb` does not.
  - `ids`, `names`, `subnetIds`, `vpcIds`, `status` are **ignored** in every
    encoding tried (flat, `filter.`-prefixed, `ids[]`, repeated params) —
    identical to sending a parameter that does not exist.
  - **Pagination is ignored** on both `GET /loadBalancers` and `GET /cas`, even
    though the spec declares `page`/`size` on `/cas` with defaults 1/10. The
    full list comes back and the envelope reports
    `page=0, pageSize=0, totalPage=0`, so `totalItem` is just the response
    length, not a total.

  The CLI therefore only offers `--name` and `--zone-id`; the rest were removed
  rather than left as flags that silently do nothing, and a test in
  `loadbalancer/builders_test.go` pins that. Re-add them in the flat form (which
  `name` proves is the right shape) once the backend implements them.
- **Response envelopes vary — five shapes, not three.** Single resources come
  back as `{"data": {...}}`, list endpoints under a load balancer as
  `{"data": [...]}`, and the paginated top-level lists (`loadBalancers`, `cas`,
  `packages`) as `{"listData": [...], "page": n, "totalItem": n}`. Beyond those:
  `list-scale-history` and `list-by-subnet` return a **bare array**, and
  `get-clone-metadata` and `GET /cas/{caId}` return a **bare object** — while
  `POST /cas` wraps in `{"data": ...}`, so `certificate get --query uuid` works
  but `certificate create` needs `data.uuid`. Each package has an `unwrapData`
  helper; completers pass the right envelope key to `cli.ExtractCompletions`.
- **`POST` keys the new id under `uuid`**, not a resource-named key:
  `lb create` returns `{"uuid": "lb-..."}`. Same for pool, listener and policy.
- **`zoneId` is `null` in `lb list`** for every load balancer even though
  `--zone-id` filters correctly server-side; `lb get` returns a full `zone`
  object instead.
- **`displayStatus` casing differs per resource**: load balancer `ACTIVE`,
  listener/pool/member `HEALTHY`, health monitor `active`. Compare
  case-insensitively, per resource.
- **`CREATING-BILLING` is a fourth transient load-balancer status**, seen
  between `CREATING` and `ACTIVE`. `evaluateActive` survives it only because it
  treats unrecognised statuses as keep-polling; do not turn that into an
  allowlist.
- **PUTs are not partial.** `updateListener`, `updatePool` and `updateL7Policy`
  declare required fields (the three listener timeouts; the pool algorithm plus
  four health-monitor thresholds; the policy action). Those commands therefore
  **GET the current resource, seed the body from it, and overlay only the flags
  the user set** — see the `buildUpdateBody` functions. Read-only DTO fields
  (`uuid`, `progressStatus`, `displayStatus`, `healthCheckProtocol`) must be
  stripped before echoing state back; tests assert this.
- `PUT .../pools/{id}/members` **replaces the entire member list** — it is not
  an append. `grn vlb pool member update` therefore carries `--dry-run` +
  `--force` and offers `--keep-existing` to merge instead of replace.
- **One mutation at a time.** Any create/update/delete puts the load balancer
  into `UPDATING`, and the next mutation fails with
  `400 The load balancer id ... is not ready`. Callers must poll
  `displayStatus` back to `ACTIVE` between operations — provisioning a new load
  balancer takes minutes. `grn vlb loadbalancer wait-active` exists for exactly
  this — put it between mutations in any script.

### Undocumented required fields (found by exercising the live API)

The spec understates what the backend demands. Breaking these rules produces
either an HTTP 500 or a 400 naming a field the user never supplied, so
`pool/create.go` validates them client-side (`validateHealthMonitor`,
`validateMembers`) and reports one actionable message instead:

- **Health monitor**: `timeout`, `interval`, `healthyThreshold` and
  `unhealthyThreshold` are always required. Omitting any of them yields
  **HTTP 500**, not a validation error. `CreateHMonitorRequest` also bounds them:
  thresholds 2-10, `timeout` 2-120s, `interval` 5-3600s. The CLI does not check
  those ranges, so an out-of-range value is answered by the API.
- **An HTTP health monitor** additionally requires `healthCheckPath`,
  `healthCheckMethod`, `successCode` and `httpVersion`
  (`CreatePoolValidator.validateHealthCheck`). It is keyed on the *health check*
  protocol, not the pool protocol. **HTTPS requires only `healthCheckMethod`**:
  `validateMethod` rejects a nil method for both, but `validateHttpVersion`
  returns early on nil and only the HTTP branch demands path/successCode. The
  CLI used to require all four for HTTPS and so refused requests the API
  accepts.
- `httpVersion=1.1` **requires** `domainName`; `httpVersion=1.0` **forbids** it.
- **A TCP or PING-UDP monitor forbids** `healthCheckPath`, `healthCheckMethod`,
  `successCode`, `domainName` and `httpVersion` — sending any of them is a 400.
- **`timeout` must be strictly less than `interval`** ("Timeout must be less
  than interval"), checked on create only — `UpdatePoolValidator` does not
  re-check it, so neither does the CLI.
- **A UDP pool must pair with a `PING-UDP` health check**, and `PING-UDP` is
  valid only for a UDP pool (`validateHealthCheckProtocol`). The rule holds
  without knowing the load balancer type, so the CLI enforces it.
- `healthCheckMethod` must be `GET`/`POST`/`PUT`; `httpVersion` must be
  `1.0`/`1.1` when present.

  All of the above are enforced client-side by `validateHealthMonitor` and
  `validateMonitorFieldCombinations` (`pool/create.go`), the latter reused by
  `pool update` against the monitor's *stored* protocol, since the backend
  re-runs the same check there. What the CLI deliberately leaves to the API:
  the numeric ranges, and which health-check protocols each **load balancer
  type** allows (L7: `TCP`/`HTTP`; L4: `TCP`/`PING-UDP`/`HTTP`/`HTTPS`) — that
  needs the LB type, which `pool create` does not fetch.
- **Member `monitorPort` is mandatory**, though the spec lists only
  `ipAddress`, `port` and `backup` as required. Omitting it returns
  `400 members[N].monitorPort: Invalid port number` — baffling, since the user
  never set one. Members read back from the API and resent via
  `--keep-existing` are exempt (they carry the service's own value).
- **`GET /pools/{id}` omits `healthMonitor` and `members`**, though `PoolDto`
  declares both — only 11 scalar fields come back. `PUT` on the pool then
  demands a *complete* monitor (`400 If healthCheckProtocol field is HTTP, the
  following fields must be specified: ...`), so `pool update` fetches the
  monitor from `GET /pools/{id}/healthMonitor` and merges the user's overrides
  into that. Seeding it from the pool payload silently produced an empty monitor
  and every pool update failed.
- **TLS fields are rejected on a non-HTTPS listener** — but the service still
  reports them on one, as `null`s and empty arrays. Echoing its own state back
  therefore fails with `400 Cannot specify client certificate, certificate
  authorities, default certificate authority, or tls security policy for http
  listener`. `listener update` skips those fields unless the listener is HTTPS
  or the user set the flag explicitly, and never echoes an empty list.
- **An L7 policy must have at least one rule**, though the spec marks `rules`
  optional. Both omitting the property and sending an empty list return
  `400 Missing required rules property` — an empty array is not enough, so the
  CLI requires `--rule` on create. There is no catch-all policy; the listener's
  `--default-pool-id`/`--default-action` handle unmatched traffic.
- **`defaultPoolName` IS populated** when a default pool is attached — it is
  `null` only when there is none. Verified live 2026-09-08 against an
  Application load balancer whose listener had a default pool attached: the
  service returned the pool's name, not null. An earlier note here
  claimed it was always null; that was wrong, and the listener table still shows
  the unreadable `defaultPoolId` because of it (see Open defects). Note
  `listener/delete.go` already prints `defaultPoolName` in its preview.
- **`stickiness` and `tlsEncryption` are gated on the pool protocol.** Backend
  `CreatePoolValidator.validateNonPoolConfiguration` tests `!= null`, so sending
  `false` is as fatal as `true`: an HTTP pool **requires** both, a non-HTTP pool
  **forbids** both. `validatePoolProtocol` additionally restricts Layer 7 to
  `HTTP` and Layer 4 to `TCP`/`UDP`/`PROXY`. `UpdatePoolValidator` applies the
  same check using the pool's stored protocol, so echoing the service's own
  `stickiness: false` back on a TCP pool is rejected too (see Open defects).
- **Member `backup` is required.** Omitting it returns
  `400 members[0].backup: must not be null;` — the spec agrees it is required,
  but `validateMembers` does not check it (see Open defects).
- **The inline `--listener`/`--pool` on `lb create` are all-or-nothing** —
  supplying one gives `400 Both listener and pool properties must be required
  (non null) or both are not required (null)`. The inline listener also requires
  all three timeouts (the standalone `listener create` defaults them to 50/50/5;
  the inline struct has no defaults), and the inline pool requires
  `healthMonitor`, which is a nested object and therefore **impossible in the
  `k=v` shorthand** — `--pool` is JSON-only in practice (see Open defects).
- **`CREATING-BILLING` is a fourth transient status.** Observed live between
  `CREATING` and `ACTIVE`. The waiter survives it only because `evaluateActive`
  treats unrecognised statuses as keep-polling; do not turn that into a strict
  allowlist.
- **There are five response envelopes, not three.** Besides `{"data": {...}}`,
  `{"data": [...]}` and `{"listData": [...], ...}`: `list-scale-history` and
  `list-by-subnet` return a **bare array**, and `get-clone-metadata` and
  `GET /cas/{caId}` return a **bare object** — while `POST /cas` wraps in
  `{"data": ...}`, so `certificate get --query uuid` works but
  `certificate create` needs `data.uuid`.
- **`POST` responses key the new id under `uuid`** (not a resource-named key):
  `lb create` -> `{"uuid": "lb-..."}`. Same for pool/listener/policy create.
- **`zoneId` is `null` in `lb list`** for every load balancer, though
  `--zone-id` filters correctly server-side. `lb get` returns a full `zone`
  object instead. The list table cannot show a zone column.
- **`displayStatus` casing differs per resource**: load balancer `ACTIVE`,
  listener/pool/member `HEALTHY`, health monitor `active` (lowercase). Compare
  case-insensitively, per resource.
- **L7 policy `position` starts at 1**, even though `ReorderPolicyRequest.position`
  is documented as zero-based. The reorder endpoint is **not exposed by the CLI**
  at all: the single live attempt (with `position=0`) was accepted with HTTP 200,
  changed nothing, and left the load balancer in `UPDATING` for about 30 minutes
  — refusing further changes and deletion alike (`400 ... is updating`) — while
  all 49 other load balancers in the project stayed `ACTIVE`. It then recovered
  by itself and deleted normally, so this is a long stall, not a lost resource.
  One observation is not proof of causation, and a 1-based reorder was never
  successfully tested, so the endpoint is listed in
  `internal/vlbclient.unimplementedSpecPaths` with that reason rather than
  shipped. `TestPathsCoverTheSpec` fails if it ever gains a builder without the
  exception being removed.

## Command style

- VLB uses **noun-group + verb**, like vServer: `grn vlb loadbalancer create`,
  `grn vlb pool member update`. Each noun is its own Go package exporting one
  `*cobra.Command`. Keep new commands in this nested style.
- `grn vlb package` lives in the `lbpackage` package — `package` is a Go keyword,
  so the directory cannot be named after the command.
- Aliases: `loadbalancer` → `lb`, `certificate` → `ca`; `certificate create` →
  `import`.
- Struct-valued flags use `cli.ParseStructFlag(Typed)` / `cli.ParseStructFlags`
  (shorthand `k=v,k2=v2` or JSON). Because commas separate a struct's own
  fields, a **list nested inside shorthand uses `|`** (e.g.
  `sniCertificateIds=ca-1|ca-2`); JSON is available for anything more complex.

## Testing

Body construction is factored into pure `buildXBody(...)` functions so it is
testable without HTTP; the `builders_test.go` files cover them. `update.go`
files expose `addUpdateFlags(*pflag.FlagSet)` separately from `init()` so tests
can build an identical flag set without mutating the shared command — and a test
asserts every declared flag appears in the "nothing to update" list (this
already caught a missing `--tls-security-policy`).

`cmd/vlb/vlb_test.go` pins the whole command surface, the required parent-ID
flags, and which commands must offer `--dry-run`.

`cmd/vlb/dryrun_test.go` drives the **real cobra tree** with argv and asserts on
what `--dry-run` prints, covering the flag-parsing-to-body glue the builder
tests cannot reach. It only covers commands whose `--dry-run` returns before any
HTTP call; the read-modify-write updates return after a GET, so their builders
are tested with fixture state instead.

`internal/vlbclient/paths_test.go` checks every generated URL against the path
template in `openapi-vlb.json`, and `TestPathsCoverTheSpec` asserts the mapping
is exhaustive in both directions — a spec path with no builder, or a builder
with no spec path, fails the build. It skips when the spec file is absent.

`internal/vlbclient/roundtrip_test.go` runs the real HTTP client against an
`httptest` server with a stub token provider, pinning the wire format: 1-based
`page`/`size`, comma-joined array filters arriving decoded, JSON numbers and
booleans surviving as such, the three response envelopes parsing, and API errors
surfacing the server message.

**What the tests do not cover:** the `RunE` network path (resolve → request →
output). `cli.NewTokenProvider` only targets the prod/dev IAM presets, so a
command cannot be pointed at a local server without real credentials. That path
was exercised by hand against a real HCM-3 project on 2026-09-08 — a full
create/update/delete lifecycle — which is where every quirk documented above
came from.

**`tests/vlb/RUNBOOK.md` is that run, written down.** It covers all 38 commands
across happy path, validation errors, dry-run, output formats and completion,
and carries the P0 regressions for the bugs that actually happened here: the
two-GET pool update, the TLS-field skip on a non-HTTPS listener, the non-partial
listener PUT, and the three member-update modes. Run it in full and attach
`RESULTS.md` before removing the `vks_only` tag — `RELEASING.md` gates on it.

It was last run on 2026-09-08 against a real HCM-3 project. That run found five
release blockers, all since fixed — see "Live-run history" below.

## Waiters

`loadbalancer wait-active` / `wait-deleted` (`loadbalancer/wait.go`) exist
because the API serialises mutations (see above). The polling loop, progress
reporting and the exit-255-on-failure convention live in `internal/cli/waiter.go`
and are shared with vks; only the evaluators are per-service. VLB's differ from
vks's in two ways worth remembering:

- the state is `data.displayStatus`, not `status` — vks's extractor would read
  nothing, and a test pins this
- `CREATING`, `UPDATING` and `DELETING` are all transient; only `ERROR` and
  `FAILED` are terminal failures

Naming note: the command is `loadbalancer wait-active`, not vks's
`wait loadbalancer-active`, to keep VLB's noun-group-then-verb shape. The leaf
verb is still the canonical `wait`, so `cmd/conventions_test.go` is satisfied.

## Live-run history

**2026-09-08, a real HCM-3 project.** `tests/vlb/RUNBOOK.md` found five
release blockers; all five are **fixed and re-verified live** on the same
project. Keep the tests that pin them — each one shipped because nothing pinned
it.

| Defect | Fix | Pinned by |
|---|---|---|
| `pool create` sent `stickiness`/`tlsEncryption` unconditionally, so **no** pool could be created on a Layer 4 load balancer (every legal L4 protocol 400'd, and the one protocol the CLI could send was illegal on L4) | gate both keys on `protocol == "HTTP"`; refuse the flags for other protocols instead of dropping them | `pool/builders_test.go` `TestStickinessIsGatedOnThePoolProtocol` |
| `pool update` echoed the service's own `stickiness: false` back, breaking every non-HTTP pool | gate on `current["protocol"]` from `GET /pools/{id}` | same test |
| `pool member update --member ''` returned the very error prescribing it, so a pool could not be emptied | guard on `Changed("member")`, not `len(memberFlags)` — pflag collapses an empty `stringArray` occurrence to an empty slice | `member/builders_test.go` `TestEmptyMemberFlagReachesTheBuilderAsPflagDeliversIt`, which drives real pflag rather than a hand-written `[]string{""}` |
| `members[].backup` (`@NotNull` boxed Boolean) produced a 400 naming a field the user never set | fill in `false`; its default is unambiguous, unlike `monitorPort`'s | `TestMemberBackupDefaultsToFalse`, `TestMemberUpdateFillsInBackup` |
| Both inline examples in `lb create --help` always failed; `--pool` shorthand was structurally impossible | `MarkFlagsRequiredTogether("listener","pool")`, default the three inline listener timeouts to 50/50/5, document `--pool` as JSON-only | `dryrun_test.go` (required-together cases + the defaulted timeouts) |

Two table columns read fields that are null in practice; both now read the
populated one:

- the listener table shows `defaultPoolName` (the service does populate it — an
  earlier note here wrongly said it never did)
- `certificate list` has **no expiry column** at all. `expiredAt` was null on
  17 of 18 certificates and contradicted its own `notAfter` on the one that was
  set, so a column reading it rendered blank and hid that 10 had already
  expired; `notAfter` is correct but is epoch milliseconds and the output layer
  does no per-column value formatting, so it rendered as a bare integer. Both
  were worse than omitting it. Expiry is surfaced by `certificate get` and by
  `certificate delete`, whose preview formats `notAfter` and marks `(EXPIRED)`.
  `formatter.cellString` was added along the way so that large JSON numbers in
  any product's table stop rendering as `1.781232534e+12`.

## Release gate

`grn vlb` is registered in `cmd/register_vlb.go` behind `//go:build !vks_only`,
so it is built and tested in CI but excluded from the public release binary
(built with `-tags vks_only`). Remove that tag when VLB is ready to ship —
after `tests/vlb/RUNBOOK.md` has been run end to end against a real project and
its `RESULTS.md` is attached to the PR that removes it.

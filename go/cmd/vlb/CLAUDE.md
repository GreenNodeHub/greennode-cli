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
- **Response envelopes vary**: single resources come back as `{"data": {...}}`,
  list endpoints under a load balancer as `{"data": [...]}`, and the paginated
  top-level lists (`loadBalancers`, `cas`, `packages`) as
  `{"listData": [...], "page": n, "totalItem": n}`. Each package has an
  `unwrapData` helper; completers pass the right envelope key to
  `cli.ExtractCompletions`.
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
  **HTTP 500**, not a validation error.
- **HTTP/HTTPS health monitor** additionally requires `healthCheckPath`,
  `healthCheckMethod`, `successCode` and `httpVersion`.
- `httpVersion=1.1` additionally requires `domainName`.
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
- **`defaultPoolName` is always `null`** on a listener, even with a default pool
  attached — the listener table shows `defaultPoolId` instead of an always-blank
  column.
- **L7 policy `position` starts at 1**, even though `ReorderPolicyRequest.position`
  is documented as zero-based. Sending `position=0` was accepted with HTTP 200,
  changed nothing, and left the load balancer in `UPDATING` for about 30
  minutes — refusing further changes and deletion alike (`400 ... is updating`)
  — while all 49 other load balancers in the project stayed `ACTIVE`. It then
  recovered by itself and deleted normally, so this is a long stall, not a lost
  resource. `buildUpdateOrderBody` refuses positions below 1 as a result. One
  observation is not proof of causation, but 0 contradicts every position the
  API itself reports.
- `POST` responses are `{"<key>": "<id>"}` string maps, not full resources.

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
came from. Repeat that before shipping.

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

## Release gate

`grn vlb` is registered in `cmd/register_vlb.go` behind `//go:build !vks_only`,
so it is built and tested in CI but excluded from the public release binary
(built with `-tags vks_only`). Remove that tag when VLB is ready to ship.

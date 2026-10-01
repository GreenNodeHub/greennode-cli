# vDB CLI — product notes

Tier-2 notes for the `grn vdb` command group. The root `CLAUDE.md` has the
repo-wide conventions; this file records what is specific to vDB.

**Do not carry vks or vserver assumptions over — vDB differs from both.**

## Client & endpoint

- Build the client with **`vdbclient.BuildClient(cmd)`**, which returns
  `(client, error)`. It wraps `cli.NewClient(cmd, "vdb")` and attaches the
  `user-type` header.
- Endpoint: `internal/config` REGIONS key `vdb_endpoint`
  (`https://vdb-gateway.vngcloud.vn`). **HCM-3 only** — vDB is not offered in HAN,
  so that region deliberately has no `vdb_endpoint` and `GetEndpoint` errors with
  a message naming the service and region.
- **There is no project ID.** Unlike vserver, no vDB path or query takes one
  (`projectId` appears only in responses). `vdbclient` has no `ProjectID` helper
  on purpose — do not add one.

## Four products, one gateway

vDB is four separate APIs that share only a hostname:

| Prefix | Group | Command root |
|---|---|---|
| `/vdb-relational/v1` | Relational (MySQL, PostgreSQL, MariaDB) | `grn vdb relational` |
| `/vdb-memory/v1` | MemoryStore (Redis) | `grn vdb memorystore` |
| `/vdb-kafka` | Kafka | `grn vdb kafka` |
| `/vdb-postgresql/v1` | PostgreSQL Cluster | `grn vdb postgresql` (+ 7 relational endpoints — see next section) |

Two of them run PostgreSQL and are easy to confuse: `relational` is single
instances (`db-`), `postgresql` is 2-10 node clusters (`pg-`). Their flavors,
volume types and catalogs are NOT interchangeable, even where the values look
alike.

They disagree on paths and HTTP verbs for the *same* operation — deleting a
backup is `DELETE /backups/{id}/delete` in relational but `POST /backups/delete`
in memory; getting an instance is `/database-instances/id/{id}` vs
`/database-instances/{id}`; the path parameter is `instanceId` in one and
`dbInstanceId` in the other.

**Keep each group in its own subpackage and let paths be duplicated.** Share only
generic helpers (`vdbclient`), never a path builder — a "unified" path helper
across groups will be wrong for at least one of them.

## PostgreSQL Cluster spans TWO prefixes — do not guess

A `pg-` cluster is served by **21 operations: the 14 under `/vdb-postgresql/v1`
plus 7 borrowed from `/vdb-relational/v1`.** The list below is the console's own
API surface for the resource, supplied by the product team on 2026-08-13 — treat it
as the reference rather than deriving anything from the spec's grouping, which
hides the split.

Borrowed from **relational** (the pg group has no equivalent at all):

| Operation | Path |
|---|---|
| list (mixed with `db-` rows) | `GET /vdb-relational/v1/database-instances` |
| get by id | `GET /vdb-relational/v1/database-instances/id/{id}` |
| action history | `GET /vdb-relational/v1/database-instances/{id}/histories` |
| security rules read/write | `GET`+`PUT /vdb-relational/v1/database-instances/{id}/secrules` |
| reboot | `POST /vdb-relational/v1/database-instances/{id}/reboot` |
| delete | `POST /vdb-relational/v1/database-instances/{id}/delete` |

Own, under `/vdb-postgresql/v1`:

| Operation | Path |
|---|---|
| create (order flow, costs money) | `POST /cluster` |
| resize (nodes and/or volume) | `PUT /cluster/{id}/resize` |
| update settings (password, publicAccess) | `PUT /cluster/{id}/settings` |
| attach config group | `PUT /cluster/{id}/config-group` |
| volume used | `GET /cluster/{id}/volume-used` |
| catalog | `GET /cluster/datastore`, `GET /cluster/flavors?zoneId=`, `GET /cluster/volume-types?zoneId=` |
| backups | `GET /backup/backup-vdb`, `GET /backup/backup-vdb/{clusterId}/detail`, `GET /backup/backup-vdb/{clusterId}/restore-point`, `POST /backup/backup-vdb/{clusterId}/backup-now` |
| backup catalog | `GET /backup/location`, `GET /backup/policy` |

Consequences for `grn vdb postgresql` (implemented — `cmd/vdb/postgresql/`):

- **There is no `start` / `shutdown` / `resize-storage` for a cluster**, and no
  replica endpoint. Storage growth is part of `PUT /cluster/{id}/resize`.
- The group is complete on its own: `cluster list/get/list-histories/list-secrules/
  update-secrule/reboot/delete` call the **relational** paths from inside
  `cmd/vdb/postgresql/cluster`, so a user never has to know the split. Do not
  invent `/vdb-postgresql` equivalents; they do not exist.
- **Every command in the group rejects a `db-` ID** via
  `vdbclient.RequireIDWithPrefix`. This is load-bearing, not cosmetic: those
  relational paths serve both products, so without the guard
  `postgresql cluster delete --cluster-id db-…` would delete a Relational Database
  instance. Any future command that reuses a shared endpoint needs the same guard.
- `cluster list` filters to `pg-` rows **and applies `--name`/`--status`
  client-side**, because the server ignores both for cluster rows. `pageObject`
  then describes the underlying mixed listing, not the filtered result — say so in
  any doc.
- Config groups for a cluster come from relational `GET /configurations` filtered
  to `deployType == "cluster"` (`catalog list-config-groups`); the cluster product
  has no config-group listing either.
- **The backup paths take a CLUSTER id, not a backup id** — `…/backup-vdb/{id}/detail`
  is "the backup state of this cluster". Do not model it on relational's
  `/backups/detail/{backupId}`.
- Unlike relational's `update/setting` and `update/config-group`, the pg
  equivalents **do** have a documented response (`InstanceActionResult`), so they
  are not part of the 6-endpoint blind spot below.
- **Multi-AZ (spec update 2026-10-01): `netIds` takes one subnet per zone.** One
  subnet places every node in its zone — the only mode before this; several subnets,
  one per zone, spread the nodes across those zones. `cluster create` enforces the two
  inferences client-side: no repeated subnet (a repeat names one zone twice) and at
  most one subnet per node (every zone named has to receive a node). The flavors and
  volume types a Multi-AZ cluster can use are a separate list — `GET /cluster/flavors`
  and `GET /cluster/volume-types` take `?multiZone=true`, exposed as `--multi-zone` on
  `catalog list-flavors` / `list-volume-types` and sent only when set, since the API
  applies its own default when the param is absent. A multi-zone cluster reports its
  per-zone placement as `multiZoneInfos[]` on the instance get/list payloads
  (`zoneId`, `subnetId`, per-zone `privateRwIp`/`publicRwIp`/`privateRoIp`/`publicRoIp`,
  `rwPort`/`roPort`, `status`), empty/null for single-zone clusters. None of this is
  exercised live yet — the mapping is from the updated spec.
- The supplied capture writes every path with a leading `/vdb`
  (`GET /vdb/vdb-relational/v1/…`). That is a **UI-only prefix, stripped by the
  console's proxy_pass** before the request reaches the API (confirmed by the
  product team). Ignore it: paths in this repo start at `/vdb-relational` /
  `/vdb-postgresql`, as `vdb_endpoint` expects.

Live-verified so far: list, get by id, histories (2026-08-13, HCM-3). The rest of
the mapping is from the supplied list, not yet exercised.

## MemoryStore differs from Relational in paths, not just prefix

`cmd/vdb/memorystore/` is implemented (42 commands, 44/44 endpoints). Same five nouns
as relational, and the *same operation* is spelled differently almost everywhere.
Copying a relational path and swapping the prefix is wrong in these places:

| Operation | relational | memorystore |
|---|---|---|
| get instance | `/database-instances/id/{id}` | `/database-instances/{id}` |
| update settings | `/{id}/update/setting` | `/{id}/update-setting` |
| update config group | `/{id}/update/config-group` | `/{id}/update-config-group` |
| catalog root | `/database-instances/*` | `/database/*` |
| flavor codes | `…/flavor_zones/codes` | `/database/codes` |
| volume types | `…/volume/types` | `/database/volume-types` |
| backup detail | `/backups/detail/{id}` | `/backups/{id}/detail` |
| backup delete | `DELETE /backups/{id}/delete` | `POST /backups/delete` (no ID in path) |
| backups of an instance | `/backups/insId/{id}` | `/database-instances/{id}/backups` |
| config group detail | `GET /configurations/id?id=` | `GET /configurations/{id}/detail` |
| config group delete | `DELETE /configurations/delete` | `POST /configurations/delete` |
| **storage you own** | `/backup-storages/information` | **`/backup-storages`** |
| **storage you can buy** | `/backup-storages` | **`/backup-storages/packages`** |
| release storage | `/actions/deletions` | `/actions/delete` |

The last three invert: `GET /backup-storages` lists purchasable packages in relational
and owned quota in memorystore.

Product differences that shape the commands:

- **No volume at all.** `CreateMemDbInstanceRequest` has no `volumeType`/`volumeSize`,
  and there is no resize-storage endpoint — capacity is the flavor's RAM. A test
  asserts the command does not exist.
- **A master password instead of a user/database pair**: `redisPasswordEnabled` +
  `redisPassword`, no `user`, no `databases`. Two rules are enforced client-side:
  public access requires the password, and passing a password implies requiring it
  (otherwise the body sets a password and disables it in the same request — a bug this
  code shipped for one build before being caught).
- **`update-setting` needs `editRedisPassword: true`** alongside any password change.
  It is a marker, not a setting, so the CLI sets it rather than exposing a flag.
- **Zones and subnets come from the RELATIONAL catalog** — memorystore has no endpoint
  for either (product owner, 2026-08-13), so the completion keys `vdb:relational-zone`
  and `vdb:relational-subnet` are reused deliberately.
- **`/vdb-memory/v1/database/status` is OUTDATED — do not use it** (product owner,
  2026-08-13). It is the only status enumeration in the API and it has neither a
  command nor a completer; `--status` completion reads the instance listing.
- Instance IDs are `db-…`, the SAME prefix relational uses, so the prefix guard cannot
  tell the two products apart. Their listings are separate, which is what does. No
  cross-product damage was observed (the wrong product's ID 404s), but do not rely on
  the guard for that.

## Kafka shares the gateway and nothing else

`cmd/vdb/kafka/` is implemented — 36 commands over 35 of the API's 36 endpoints. It is
the group that breaks the most assumptions the other three teach:

| Assumption from the other groups | Kafka |
|---|---|
| every response is `{code, message, data}` | only 9 of 36 are wrapped; the rest are bare objects/arrays/strings |
| lists are paginated, filters are server-side | no `pageNumber`, no filters, no `--page` at all |
| mutating arguments go in a JSON body | 6 endpoints take them in the QUERY STRING of a body-less PUT |
| the prefix carries `/v1` | `/vdb-kafka` has none |
| security rules are replaced as a set | added and removed one at a time, and there is **no endpoint that lists them** |
| there are backups | none, at all |
| a config group is edited in place | groups are versioned and immutable; a cluster attaches to a *version* |
| flavors and volume types are numeric ids / names | opaque `flav-…` and `vtype-…` strings |

Consequences encoded in the code:

- **`vdbclient.Client.Request`** exists for the query-string PUTs (`kafka-broker-count`,
  `kafka-storage-size`, `kafka-storage-type`, `authentication`, `config-group`,
  `public-access`). `Put` takes no params and `Delete` takes no body, so neither fits.
  `--dry-run` for those prints the QUERY, via `previewQuery` in `kafka/cluster/helpers.go`
  — `vdbclient.PreviewBody` would print `null`.
- **`vdbclient.Client.NoContent`** is used by the 11 mutating endpoints the spec declares
  as a bare `string`. Live (2026-08-17) they answer 200/204 with an **empty body**, which
  would print as `{}`. Per the product decision the HTTP status is the whole result: those
  commands print their own confirmation and name the read command that shows the effect.
- **`cluster list-secrules` has no endpoint** — it reads the cluster and prints the nested
  `securityGroupRules`. Note the LISTING omits that field even though `KafkaCluster`
  declares it; only `GET /clusters/{id}` carries it (verified live).
- **`GET /vdb-kafka/database/configs` has no command** (product decision: it is the
  service's own settings). It is read internally anyway, because it is the ONLY place the
  Kafka version list exists — `kafkaVersions` is a JSON array encoded inside a string,
  since the payload is a flat `map<string,string>`. `catalog/completion.go` decodes it
  twice. The same payload also confirms the client-side bounds this group enforces:
  `minKafkaBrokers` 3 / `maxKafkaBrokers` 10, `maxTopicPartitions` 2048,
  `minTopicRetentionHours` 1 / `maxTopicRetentionHours` 2160,
  `secGroupRulePorts` `[9092,9094,9096,9194,9196]`.
- **`serverFlavorId` takes `flavorId`, `kafkaStorageType` takes `kafkaUuid`** — both
  verified live against a running cluster. Not the numeric `id`, and not the display
  `type` (`kafka.Gen2-NVMe2-IOPS3000`). Relational Database is the opposite on both counts.
- **`GET /database/volume-types` is the one catalog endpoint whose envelope does not hold
  a bare array**: `data` is `{projectId, data: [...]}`. `catalog.go` strips the second
  layer (`volumeTypeItemsKey`) for both output and completion.
- **The flavors `version` filter is accepted and ignored** (3.7.0 and 9.9.9 both return
  all 24 rows, and the rows carry no version field), while `type` really filters and is
  case-insensitive (`kafka`/`Kafka` fine, `bogus` → 0). `--datastore-type` is the one
  hard-coded `cli.FlagValues` in the vDB tree, for want of an endpoint — the value was
  measured, not assumed.
- **Every "update" that replaces a set reads the resource first**: `topic update`,
  `user update` and `cluster update-authentication` all take the FULL set on every call,
  so an unset flag means "keep the current value". `topic update` additionally OMITS a
  retention field that is null on the topic and unset on the command line — sending 0
  would be an invalid value, not an absence.
- **Statuses are their own vocabulary**: `WAITING_CREATING`, `CREATING`, `ACTIVE`,
  `WAITING_UPDATING`, `UPDATING`, `DELETING`. Operations are refused while a resource is
  busy, with a clear message (`Cluster … is not active`, `Topic … does not have
  appropriate status`) — so anything scripted must poll.
- **`createdAt` differs by endpoint, as elsewhere in vDB**: listings return a localized
  display string (`Aug 17, 2026, 10:05:03 AM`, GMT+7), detail endpoints ISO-8601 UTC.
  Pass through unchanged; never compare across the two.
- **No prefix guard.** `RequireIDWithPrefix` is deliberately not used: no Kafka path is
  shared with another product, so there is nothing for a guard to protect against, and the
  ID formats (`clus-`, `topic-`, `user-`, `cgroup-`, `cgroupver-`) were confirmed only
  from one project's data.

Live-verified 2026-08-17 on HCM-3: every read command, every completer, and the full
topic write path (create → get → update → delete, including the busy-status refusals).
The cluster-level order flows (create, the three resizes) and the destructive cluster
commands were exercised only with `--dry-run`.

## API quirks

- **`update/setting` and `update-setting` reject a partial body: `backupAuto` is
  mandatory.** Verified live on 2026-08-13 in BOTH relational and memorystore: a body of
  `{dbInstanceId, redisPassword, editRedisPassword, redisPasswordEnabled}` answers
  **HTTP 500**, and so does one carrying only `publicAccess`; add `backupAuto` (plus
  `backupTime`/`backupDuration` when it is true) and the same request succeeds. Both
  `settingsBody` implementations therefore take the current instance record and repeat its
  schedule when the user named no backup flag. Do not "simplify" that back to
  changed-fields-only. The pg `PUT /{id}/settings` endpoint is a different request class
  with no backup fields and is unaffected.
- **`instanceCount` on a config group is always 0**, in all three products, even for a
  group with an instance attached (live, 2026-08-13). It is excluded from every column set
  for that reason. The truth is `configuration get` → `instances[]`.
- **An attached config group shows up as `configuration: {id, name}` on the instance,
  while the sibling `configId` stays null.** Read `configuration`, never `configId`. A
  detach leaves the instance `RESTART_REQUIRED` until a reboot.
- **"Accepted" is not "applied", and a second call while the first runs is silently
  dropped.** The async endpoints return 202 regardless; if the instance is mid-change the
  work is discarded and only `list-histories` shows it, as a `Failed` row with
  `Cannot perform action EDIT, database [...] status is BUILDING`. Anything scripted must
  poll for `ACTIVE` between changes. This is also why `warnIfNotActive` exists.
- **Master-password rules differ per product** (`internal/vdbclient/credential.go`):
  8–32 characters for relational, 16–128 for memorystore, and in both only letters,
  digits and `$ ^ _ < >`. MemoryStore additionally refuses a password containing BOTH
  `<` and `>` — either alone is fine (live probe). pg stays deliberately unchecked
  client-side: the spec documents its rules as of 2026-10-01 (8–32, the same charset,
  must start with a letter and end with a letter or digit), but they are not
  live-verified — and the start/end rule is not enforced for relational either, whose
  live probes never covered it — so the API's own message, surfaced in full by the
  error layer, remains the check.
- **A rejected password comes back in the error message in plain text**
  (`Redis password [hunter2…] contains invalid character`). `enrich` takes the request
  body so it can mask any value the body carried under a sensitive key before the message
  is printed. Keep passing the body when adding a client method.
- **`action` in a history entry does not identify the action; `description` does.**
  Settings changes, resizes and config-group attach/detach all record as `Update`, and
  only `description` says which (live, 2026-08-13). It is the second column of
  `historyColumns` in all three products for that reason — an earlier version left it out
  as "too long", which made the table useless for exactly the rows people look up.
  `updatedTime` went the other way: it equalled `createdTime` on all 57 records seen, so
  it is JSON-only now. Tests assert both.
- **A flavor name can appear twice in one zone**, once with `familyType`/`platformType`
  populated and once with both null (e.g. `db.s-general-2x4` = id 179 with
  `general-purpose`/`code-s`, id 227 with nulls). They are different package ids. When
  looking one up for `--package-id`, filter on `platformType`, not the name alone.
- **Response envelope**: relational / memorystore / postgresql wrap every payload
  as `{code, message, data}` — all 42 `Wrap*` schemas in the spec have exactly
  those three properties. `vdbclient.Output` / `OutputWithColumns` call
  `vdbclient.Unwrap` to strip that layer.
  Kafka is mixed: only 9 of its 36 operations are wrapped, the rest return bare
  objects, bare arrays or plain strings. `Unwrap` passes those through untouched,
  so calling it is safe — just never assume a kafka response was wrapped.
- **The payload inside the envelope varies per endpoint — look it up, do not
  generalise.** Across the 112 wrapped operations:

  | Payload shape | Endpoints |
  |---|---|
  | bare array | 76 |
  | the resource object itself | 17 |
  | paginated list (`{items[], pageObject}`) | 8 |
  | **spec declares no shape at all** | 6 |
  | unpaginated list wrapper | 3 |
  | `map<string,string>` | 1 |
  | plain string | 1 |

- **Paginated lists use two different item keys.** Only 2 of the 8 use `data`
  (`GET /database-instances`); the other 6 use **`content`** (`/backups`,
  `/configurations`, `/histories`). A `--query` example or doc page written for
  one does not transfer to the other.
- **6 endpoints take a JSON ARRAY as the request body**, not an object — both
  `delete` endpoints for backups and configurations, and both `secrules` updates.
  Send a slice.
- **The action endpoints share ONE request class, but each accepts only its own
  action.** `start`, `stop`, `reboot`, `detach-replica` and `delete` all take
  `{databaseInstances[], action, resType}` with the ID repeated in the body, and the
  spec lists every action value the family knows
  (`start|stop|reboot|detach_replica`) on each of them. That is a spec artifact:
  send the action matching the path and nothing else (confirmed by the product
  team). `resType` is always `dbaas`.
  Same story for the per-instance `config` on delete: the schema offers
  `createFinalBackup` and `deleteAllBackup`, but **a PostgreSQL Cluster honours only
  `createFinalBackup`** — `deleteAllBackup` is for Relational Database instances.
  Do not read a shared schema as "everything applies everywhere".
- **6 endpoints have no documented response shape in the spec** — it declares
  `data: {"type": "object"}` with no properties. They are the *same three operations
  in each of the two instance families*. **All six are resolved**: real responses
  supplied by the product team on 2026-08-13, recorded below. Use these; do not
  re-derive them from the spec, which still says nothing.

  | Operation | Relational | MemoryStore |
  |---|---|---|
  | list replicas | `GET /database-instances/{id}/replicas` | `GET /database-instances/{id}/replicas` |
  | update setting | `PUT /database-instances/{id}/update/setting` | `PUT /database-instances/{id}/update-setting` |
  | update config group | `PUT /database-instances/{id}/update/config-group` | `PUT /database-instances/{id}/update-config-group` |

  **`/replicas` → `data` is a BARE ARRAY** of replica objects, identical in both
  families. Each item is a *reduced* instance: 19 fields, not the ~68 of a full one
  — `id`, `name`, `status`, `ram`, `vcpus`, `datastoreType`, `engineGroup`,
  `datastoreVersion`, `volumeType`, `volumeSize`, `created`, `updated`,
  `publicAccess`, `backupAuto`, `volumeTypeZoneId`, `quotaPackageId`, `poc`,
  `bandwidth`, `enableAutoRenew`.
  Two consequences: **there is no `ip`, `port` or `securityGroup`**, so the instance
  `list` column set cannot be reused as-is; and since the items carry no nested
  arrays at all, `OutputWithColumns` is unambiguous here.

  **The four `update-*` endpoints share one response**, and all four are
  **asynchronous**:

  ```jsonc
  {"code": 200, "message": "success",
   "data": {"status": 202, "dbInstanceId": "...", "projectId": "..."}}
  ```

  Treat a successful HTTP status as "accepted", not "applied" — `data.status` is
  202 while the envelope `code` is 200 — and poll `instance get` for the effect.
  Note this is NOT the `InstanceActionResult` that the PostgreSQL Cluster
  equivalents return (no `success`, `action` or `errorMsg` field), which is why
  copying the sibling shape would have been wrong.
  **Ignore `data` entirely on these four.** `dbInstanceId` and `projectId` come back
  transposed (the project ID sits in `dbInstanceId` and vice versa) and the product
  team confirmed the payload carries no meaning: the HTTP status is the whole
  result. A command here should print its own confirmation and tell the user to poll
  `instance get` — never echo this `data`.

  Note the earlier count of "7" also included kafka `GET /database/configs`; that
  one is documented after all — `map<string,string>` via `additionalProperties`,
  which the api-map script used to mislabel.
- **`created`/`updated` differ per endpoint in format AND timezone.** The instance
  listing returns `2025-09-22 08:24:18.0` — no offset, and the value is local
  GMT+7 — while `/replicas`, histories and backups return
  `2026-08-13T03:31:27.000+00:00`, i.e. UTC with an explicit offset. Measured on the
  same cluster, the two are exactly 7 hours apart, to the second.

  **Decision (product owner, 2026-08-13): these fields are display-only. Pass the
  string through unchanged.** Do not parse, convert, normalise or offer a flag for
  it — a converter would have to assume "missing offset means +07", which breaks
  silently the moment the backend or the region changes. Any command that needs to
  compare times must use the values from a single endpoint, never across two.

**Look these up rather than guessing**: `docs/superpowers/vdb-api-map.py`
regenerates `vdb-api-map.md` — every operation with its path/query/header params,
request body and response shape, read straight from the spec. That directory is
local-only and git-ignored; run the script to recreate it.
- **Pagination is 1-based**: `pageNumber` (from **1**) + `pageSize`. Same as
  vserver, opposite of vks. `--page` passes straight through — no offset. Build
  the query with `vdbclient.BuildListQuery`. `pageObject.number` in the response
  echoes the 1-based page, so it can be fed back in (verified live).
- **`filterRequest` is flattened**: the spec models list filters as a
  `filterRequest` object, but the API takes plain query params
  (`?name=x&status=ACTIVE`), with the key repeated for multiple statuses. Not JSON.
  `name` is a substring match. Both verified live.
- **`GET /vdb-relational/v1/database-instances` is a MIXED listing.** It returns
  Relational Database instances (id prefix `db-`, `dbBackendId` set) *and*
  PostgreSQL Cluster records (id prefix `pg-`, `dbBackendId` null). That is by
  design: the PostgreSQL Cluster API has **no list and no get-by-id endpoint at
  all**, so this is the only way to enumerate clusters.
  **The `name` and `status` filters do not apply to the `pg-` rows** — they come
  back on every query, including `?name=<nonsense>` and `?status=<nonsense>`.
  Never document these filters as exact, and filter client-side by id prefix if a
  command must show only one product.
  **Seven relational endpoints serve `pg-` IDs on purpose** — get-by-id, histories,
  secrules (read + write), reboot and delete, besides this listing. A cluster comes
  back with `deployType: cluster`, `numberOfNodes` and `privateRwIp`/`publicRwIp`
  instead of an `ip` list. See the PostgreSQL Cluster section above for the full
  map; resize, settings and backups are NOT among them.
- **Latency differs by an order of magnitude.** The catalog endpoints answer in
  ~0.2s, `GET /database-instances` in ~1.3s with spikes past 6s (measured live in
  HCM-3). That matters for `cli.FlagFromAPI` completion, whose 2s bound the
  instance listing sometimes exceeds — instance-ID completion is best-effort by
  nature, catalog-backed completion is reliable.
- **`user-type` header**: `ROOT_USER` (Checkout) or `IAM_USER` (Auto Payment) on
  the ~19 chargeable endpoints. Exposed as the persistent `--user-type` flag on
  `grn vdb`; `BuildClient` sends it only when non-empty so the API keeps its own
  `ROOT_USER` default.
- **Create is an order/payment flow** (`/payment/database-instances`,
  `createOrderCluster`) — it **costs money** and completes asynchronously. Every
  `create` and `resize` needs `--dry-run` plus a confirmation and `--force`, even
  though `conventions_test.go` only requires them for delete/stop/reboot.
- **Mutation response shapes, all verified live 2026-08-13** (relational):

  | Endpoint | Response |
  |---|---|
  | create, create-replicas | `OrderResponse[]` — `resourceId` is the NEW instance ID |
  | resize-instance, resize-storage | `OrderResponse[]` with **`resourceId: null`** |
  | start, stop, reboot | `ActionDbInstancesResponse[]` — `code: 202`, `success: true`, `status: PROCESSING` |
  | delete, detach-replica | same shape but **`code: null`, `success: null`** — do not test `success == true` |
  | update/setting, update/config-group | meaningless body, see below |
  | secrules PUT | `SecurityGroupRuleEntity[]`, the resulting rule set |

  Note `databaseInstances` is a plain STRING (the instance ID) in an action
  *response* while it is an array of objects in the *request*.
- **"Optional" fields that are effectively required, all found by running the real
  API (2026-08-13). The pattern is the same each time: the request is accepted, then
  the work fails in the background — so the spec cannot be trusted here and neither
  can a 200.**

  | Endpoint | Field the spec calls optional | What happens without it |
  |---|---|---|
  | `POST /backups/create` | `description` | 200 + `success:true` + a backup ID, then the backup fails with *"An error occurred when communicating with system"*. 4 failures out of 4 (including `description: ""`), 2 successes out of 2 with a non-empty one. |
  | `POST /backups/{id}/restore` | `locateZoneId` | HTTP 400 blaming the *flavor and volume type*: `Package ID 180 is invalid; Volume type Gen2-NVMe2-IOPS3000-HCM03-1B is invalid` — even when those are exactly what the source instance runs. The zone is the real cause. |

  Both are enforced client-side so the error names the actual problem.
- **`PUT /configurations/update` REPLACES the whole `values` map.** Setting one
  parameter drops every other parameter the group held (verified live). The CLI shows
  before/after and offers `--merge`. It is also asynchronous: a read straight
  afterwards still returns the old map.
- **"Not found" can arrive as HTTP 200 with `data: null`.** A backup whose creation
  failed answers `{"code":200,"message":"success","data":null}`. `Unwrap` hands back
  the envelope in that case, so a caller reading fields off it sees empty strings and
  zeroes — which is how a `restore` would end up sending a request built from
  nothing. **Any command that fetches one resource and reads fields from it must use
  `vdbclient.PayloadObject` and report the miss.**
- **Backup storage: the free allowance is per-instance, and `resourceId` behaves
  differently from an instance resize.** Verified live 2026-08-13 by buying, resizing
  and releasing a package: `freeBackupStorage` is the SUM of every instance's
  `freeBackupSize` (100 GB for a 4x8 flavor), so it changed 300 -> 400 -> 300 as an
  instance appeared and went; buying returns `OrderResponse[]` whose `resourceId` is
  the new `db-bk-storage-…`; and unlike an instance resize, the storage resize DOES
  return a `resourceId`. Both create and resize applied within seconds, not minutes.
  `backupPackageName` is null in the listing even right after a purchase.
- **A resize to the value it already has is accepted, then fails asynchronously.** No
  4xx: the order goes through and the history records `Update ... Failed` with an
  empty change description. That is why `resize-storage` refuses a no-op
  client-side — the API will not tell you.
- **Several endpoints demand fields they could have defaulted themselves.** Where
  that happens, read the current state and fill them in rather than making the user
  retype them — and say so in `--dry-run`:
  `relational resize-storage` must send size AND type on every request ("if
  unchanged, use the current value"), so it GETs the instance first;
  `relational create-replica` requires a dozen fields that all describe the source,
  so it defaults every one of them to it. Two field-name traps there: the flavor is
  `quotaPackageId` on an instance but `packageId` in a request, and an attached
  config group appears only in the nested `configuration.id` — the flat `configId`
  is null (verified live).
- **`--package-id` and volume types are spelled differently per product.**
  Relational takes the flavor's **numeric id** and the volume type **NAME**;
  PostgreSQL Cluster takes opaque IDs (`pgp-…`, `pgst-…`). Passing one product's
  values to the other is rejected. Two details verified by a real create
  (2026-08-13):
    - `packageId` is accepted as a **JSON string** (`"179"`), which is what the CLI
      sends, even though the console sends it as a number. Both work.
    - **The volume type name is zone-dependent.** `Gen2-NVMe2-IOPS3000` in
      HCM03-1A but `Gen2-NVMe2-IOPS3000-HCM03-1B` in HCM03-1B — the zone is part of
      the name in some zones and not others. Always take the string from
      `catalog list-volume-types --zone-id <the zone you are creating in>`; never
      build it by hand or reuse one zone's value in another.
    - Omitting `configId` works; the console sends `""` instead. Both mean
      "no config group" **on create** — but not on update, where `""` detaches.
- **Automatic backup is a set of three fields**: `backupAuto` plus, when it is on,
  `backupDuration` (2-14 days) and `backupTime` (`HH:MM`). Send all three together
  or none; the CLI validates the combination client-side so the error names the
  flag rather than coming back as a 400.
- **`update/config-group` detaches on `null`, NOT on `""` — and the two families
  disagree.** The spec says "set to empty string to detach" for both. Verified live
  2026-08-13:

  | Group | Detach payload | Result |
  |---|---|---|
  | relational | `{"configId": ""}` | **HTTP 400** `The configId  doesn't exist` |
  | relational | `{"configId": null}` | detaches, instance becomes RESTART_REQUIRED |
  | postgresql | `{"configGroupId": ""}` | accepted, `success: true` |

  This is the clearest case yet for the rule against extrapolating between families:
  fixing relational to `null` and "helpfully" doing the same to postgresql would have
  broken a working command.
- **These async updates can be accepted and silently dropped.** While an instance is
  RESTART_REQUIRED, `update/config-group` answers 202 `success` and never applies the
  change — an attach was issued, acknowledged and lost (verified live). The change
  landed only once the instance was ACTIVE again. `warnIfNotActive` in
  `relational/instance/async.go` warns before sending; nothing else can detect it,
  since the API reports success either way.
- **Status values are mixed-case and include more states than the spec implies.**
  Seen live: `BUILDING`, `ACTIVE`, `RESTART_REQUIRED`, `SHUTDOWN`, and lowercase
  `stopping`, `starting`, `deleting`; action responses carry `PROCESSING`. **Compare
  case-insensitively.**
- **A deleting instance can report ACTIVE again before it disappears.** Observed
  `deleting` → `ACTIVE` → HTTP 404. A waiter must treat only the 404 as "deleted";
  keying on ACTIVE would call a deletion "finished" as a success.
- **A BUILDING instance reads back defaults, not what you sent.** During
  provisioning `get` reported `backupAuto: false` with null duration and time, and a
  null `ip`, for an instance created with backup enabled; all three appeared
  correctly the moment status became ACTIVE (verified end-to-end, 2026-08-13, ~5
  minutes). So do not diff a create request against a BUILDING instance and conclude
  the API ignored a field — wait for ACTIVE. Anything that verifies or waits must key
  on `status`.
- **No enums in the spec**: status, datastore type, volume type and flavor are
  free-form strings. Shell completion must use `cli.FlagFromAPI` against the
  catalog endpoints (`/database/datastore`, `/database/volume-types`,
  `/database/flavors`), not hard-coded `cli.FlagValues`.
  **`/vdb-memory/v1/database/status` is the one status enumeration in the API, and it
  is OUTDATED — do not use it** (product owner, 2026-08-13). Both groups derive
  `--status` completion from their own instance listing instead, which cannot go
  stale.
- **Case of `datastoreType` is not consistent.** `/database-instances/datastore`
  reports lowercase (`postgresql`), instance payloads report display form
  (`PostgreSQL`), and the `/flavors` query accepts either (verified live).

## Command style

- **Engine group + noun group + verb**: `grn vdb relational instance list`,
  `grn vdb kafka topic create`. Chosen over a single `--engine` flag because the
  four APIs share no shape; see the table above.
- Package layout mirrors the tree: `cmd/vdb/<group>/<noun>/<verb>.go`.
- Rename these API operations to canonical verbs: `updateSecurityRules` →
  **`update-secrule`**, `updateSettings`/`updateDatabaseSetting` →
  **`update-settings`** and `update*ConfigGroup` → **`update-config-group`** in every
  group. The API paths disagree on singular vs plural (`/settings` for pg,
  `/update/setting` for relational); the CLI does not follow them there — one verb
  per operation across products is the point of this list.
  Kafka's `regenerateUserAuthenCredential`
  (`PUT /vdb-kafka/clusters/{id}/users/{userId}/regenerate-creds`) is
  **`generate-creds`** (`regenerate` is not one of the canonical verbs), and
  `getUserAuthenCredential` is **`get-creds`**.
  Kafka's three chargeable PUTs keep the verb that says what they cost:
  `kafka-broker-count` → **`resize-brokers`**, `kafka-storage-size` →
  **`resize-storage`** (matching relational, though Kafka's carries no type), and
  `kafka-storage-type` → **`update-volume-type`**.
- **Credentials are a Kafka-only concept.** Kafka has
  `GET /clusters/{id}/users/{userId}/authen-creds`, the regenerate above, and
  `PUT /clusters/{id}/authentication`. **MemoryStore has no user or credential
  endpoint at all** — a Redis password is changed through fields of its
  `update-setting` request (`editRedisPassword`, `redisPasswordEnabled`,
  `redisPassword`), so it is part of `update-settings`, not a command of its own.
  (Corrected by the product owner on 2026-08-13 after I attributed `generate-creds`
  to MemoryStore.)
- Mutating commands are gated beyond what `conventions_test.go` requires: the
  order-flow ones (`create`, `resize`) and anything that can break connectivity
  (`update-settings`, `update-config-group`, `update-secrule`) carry `--dry-run` +
  `--force` + a confirmation, alongside the mandated `delete`/`reboot`.
- A `--dry-run` that prints a request body must **mask secrets** — see
  `postgresql/cluster/preview.go`. Master passwords travel in create and
  update-settings bodies, and dry-run output gets pasted into tickets.
- Master passwords are read from **`$GRN_VDB_MASTER_PASSWORD`** when `--password`
  is omitted, so they stay out of shell history.
- The read-only lookup endpoints live under one **`catalog`** noun
  (`catalog list-flavors`, `list-zones`, ...) rather than a noun each. They share a
  shape — GET, no filter beyond an optional zone, a bare array in the envelope —
  and one purpose: what you consult before creating an instance, and what
  completion reads. The eight that need no arguments are declared in a table in
  `catalog/catalog.go`; only `list-flavors`, which takes required flags, has its
  own file.

## Shell completion

**Bind a flag to its completer in the `init()` of the file that DEFINES the
flag** — never from one central `registerCompletions()` in the package's parent
file, the way `cmd/vks/completion.go` does it.

Within a package Go runs `init()` functions file by file in filename order, so a
central binding called from `instance.go` runs before `list.go` and
`list_histories.go` have added their flags. `RegisterFlagCompletionFunc` then
fails on the unknown flag, and the error is discarded at init time (it is the
`//nolint:errcheck` line everyone copies), so the completion looks wired and
silently does nothing. This actually happened here. Keep the completer functions
in `completion.go` and the bindings next to the flags;
`completion_test.go` / `catalog_test.go` assert each binding via
`cmd.GetFlagCompletionFunc`.

Cross-package values go through the `cli` resource registry instead of an import,
so registration lives next to the endpoint that owns the shape and dispatch happens
at completion time (no init ordering involved). Registered today:

| Key | Source |
|---|---|
| `vdb:relational-instance` | relational instance listing (all IDs, `db-` and `pg-`) |
| `vdb:relational-datastore-type`, `vdb:relational-datastore-version` | `/database-instances/datastore` |
| `vdb:relational-zone` | `/database-instances/zones` (project-wide; the pg group reuses it) |
| `vdb:relational-volume-type` | `/database-instances/volume/types` |
| `vdb:relational-subnet` | `/database-instances/networks/subnets`, one level down into the nested `subnets` |
| `vdb:relational-network` | `/database-instances/networks` (project-wide; Kafka's `--network-id` reuses it) |
| `vdb:relational-config-group` | `/database-instances/configuration`, minus the `cluster` deploy types |
| `vdb:relational-flavor` | `/database-instances/flavors` — **context-dependent**: reads `--datastore-type`/`--datastore-version` (and `--zone-id`) off the command being completed, and stays silent until they are set |
| `vdb:memorystore-instance`, `vdb:memorystore-datastore-type`, `vdb:memorystore-datastore-version`, `vdb:memorystore-config-group` | `/vdb-memory/v1/database*` (zones and subnets reuse the relational keys) |
| `vdb:memorystore-flavor` | `/database/flavors` — context-dependent like its relational twin, but on `--datastore-version` alone; `--datastore-type` defaults to Redis |
| `vdb:postgresql-cluster` | relational listing filtered to `pg-` |
| `vdb:postgresql-datastore-version`, `vdb:postgresql-flavor`, `vdb:postgresql-volume-type` | `/vdb-postgresql/v1/cluster/*` |
| `vdb:postgresql-config-group` | relational `/configurations` filtered to `deployType: cluster` |
| `vdb:postgresql-backup-location`, `vdb:postgresql-backup-policy` | `/vdb-postgresql/v1/backup/{location,policy}` |
| `vdb:kafka-cluster` | `/vdb-kafka/clusters` (the whole listing — no pagination to page through) |
| `vdb:kafka-secrule` | the `securityGroupRules` nested in `GET /clusters/{id}` — **context-dependent** on `--cluster-id`, since there is no listing endpoint |
| `vdb:kafka-topic`, `vdb:kafka-user` | `/clusters/{id}/{topics,users}` — context-dependent on `--cluster-id` |
| `vdb:kafka-topic-name` | the same topic listing, by **name**: the user permission flags take names, not IDs |
| `vdb:kafka-config-group`, `vdb:kafka-config-group-version` | `/vdb-kafka/config-groups`; the version key digs into each group's nested `versions`, narrowing to `--config-group-id` when that flag is set |
| `vdb:kafka-flavor`, `vdb:kafka-volume-type` | `/vdb-kafka/database/{flavors,volume-types}` — `flavorId` and `kafkaUuid`, NOT the numeric ids |
| `vdb:kafka-version` | `/vdb-kafka/database/configs` → `kafkaVersions`, the only source in the API |

Consume with `cli.ResourceCompletion(<key>)`. Keep the per-product keys distinct even
where they look duplicated — the values are not interchangeable.

Not every completer needs a key: `backup-storage`'s `--package-id` and `--storage-id` are
consumed only inside their own package, so they are plain local completers in
`backupstorage/completion.go`. Note which path feeds which there — `GET /backup-storages`
means "packages to buy" in relational and "storage you own" in memorystore, so a copied
completer would suggest the wrong list.

**Ids that are JSON numbers need `vdbclient.ExtractIDValues`, not `cli.ExtractIDs`.** A
flavor `id` is `179` and a backup package `packageId` is `1` — numbers, which
`cli.ExtractIDs` skips, leaving completion silently empty (that is exactly how the
backup-storage gap went unnoticed until it was tried against the live API). Verify a new
completer by running `grn __complete vdb …` rather than trusting the binding test, which
only proves a function is attached.

## Shared helpers vs per-group code

`internal/vdbclient` now holds everything two groups would otherwise copy:

| Helper | Why it is shared |
|---|---|
| `ActionBody` / `ResizeBody` / `ResizeConfigBody` | the `{databaseInstances[], action, resType}` shape; that resize spells it `resourceType`; that the resource type is `dbaas`, `dbaas-backup` or `dbaas-backup-storage` depending on the resource; and that restore's detail carries a config with NO `instancesId` |
| `Payload` / `PayloadObject` | the "HTTP 200 with `data: null` means not found" check |
| `ParseSecurityRules` / `SecurityRulesFrom` / `DescribeSecurityRules` / `SecurityRuleColumns` | identical between relational and memorystore; only the default port differs, so it is a parameter |
| `PreviewBody` | the `--dry-run` body preview, with secret masking |
| `RequireIDWithPrefix` | the `db-`/`pg-` guard |
| `Unwrap`, `BuildListQuery`, `Get`, `Output*`, `Client` | from PR 1 |

**Still never shared: path builders.** Each group keeps its own path constants even
where the strings look alike — that asymmetry is the whole reason the groups are
separate packages.

## Errors

**Build the client with `vdbclient.BuildClient`, which returns a
`*vdbclient.Client`, not the shared `*client.GreennodeClient`.** The wrapper exists
for one reason: vDB puts a machine code in the error body's `message` and the
sentence that explains the problem in `errors[].message`:

```jsonc
{"code":400, "message":"in_valid",
 "errors":[{"typeError":"pageSize","fieldError":"invalid",
            "message":"The field pageSize of the request must be greater than 0"}]}
```

The shared `formatError` stops at the first recognised field, so it always picks
`message` and the useful part was visible only under `--debug`. `Client` runs every
response through `enrich` (`internal/vdbclient/error.go`), which appends the
details:

```
API error (HTTP 400 Bad Request): in_valid: The field pageSize of the request must be greater than 0 (pageSize)
```

The machine code is kept — support greps for it. `*vdbclient.Error` embeds and
unwraps to `*client.APIError`, so `errors.As` still reaches `StatusCode`/`Body` for
waiters. An error with no usable details, or one that is not an API error, passes
through untouched. **Do not "fix" this in `internal/client`** — vks and vserver
depend on its current message format.

`enrich` also redacts: it receives the request body from every `Client` method that has
one, collects the values under `sensitiveKeys` (shared with the `--dry-run` preview) and
replaces them in the message. That closes a real leak — the API echoes a rejected
password verbatim. Values shorter than 4 characters are ignored, so a toy password cannot
blank out unrelated text. The raw `APIError.Body` is left alone; `--debug` still prints
what the server actually said.

Also worth knowing: the async `update-*` endpoints answer 200 with a `data` payload
whose fields are unreliable (see the transposed IDs above). For those, the HTTP
status is the only thing to trust — the product team confirmed the body carries no
meaning. Print a confirmation, not the payload.

## Output

- Table output in `vdbclient` converts every number to a plain decimal string
  first (`plainNumbers`). JSON numbers decode to float64 and the table formatter
  prints with `%v`, which renders a 5861298-byte backup as `5.861298e+06`. JSON and
  text output keep the original numbers, so `--query` arithmetic is unaffected.
  This is deliberately vdb-local: the shared formatter is left alone.
- List items are very wide (a database instance carries ~68 fields, billing
  metadata included), so list commands **must** pass a column set to
  `vdbclient.OutputWithColumns`. JSON output is unaffected and keeps every field.
- **`OutputWithColumns` is for lists only.** It goes through
  `formatter.extractRows`, which returns the first slice it finds while ranging
  over the map. That is deterministic for a list payload (one array: the items),
  but a detail payload has several nested arrays — a database instance carries
  `ip`, `securityGroup`, `replicas` and `sharedActions` — and Go randomises map
  iteration, so table output would pick one of them at random. Detail commands
  (`get`) must use `vdbclient.Output`, which renders an object as a key/value
  table.

## Build gating

`cmd/vdb` is registered from `cmd/register_vdb.go` behind the `!vks_only` build
tag, so it is in dev/CI builds but excluded from the public release binary
(`-tags vks_only`) while vDB is under development. Drop the tag at GA.

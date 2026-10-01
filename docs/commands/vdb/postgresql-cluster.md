# postgresql cluster

Manage vDB PostgreSQL Clusters — the multi-node PostgreSQL product, distinct from
the single instances under [`grn vdb relational`](relational-instance.md).

```bash
grn vdb postgresql cluster <command> [options]
```

Cluster IDs start with `pg-`. A cluster has 2–10 nodes, its own flavors and volume
types, and its own backup service.

!!! note "Two APIs behind one command group"
    The cluster product has no read, reboot, delete or security-rule endpoints of
    its own — those are served by the Relational Database API, which accepts `pg-`
    IDs. Each command below says which API it calls. Because those relational
    endpoints also serve Relational Database instances, every command here
    **rejects a `db-` ID** rather than acting on the wrong product.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List clusters |
| [get](#get) | Show one cluster in full |
| [list-histories](#list-histories) | Action history of a cluster |
| [get-volume-used](#get-volume-used) | Storage in use |
| [list-secrules](#list-secrules) | Security rules |
| [create](#create) | Create a cluster (**paid order**) |
| [resize](#resize) | Change storage or node count (**paid order**) |
| [update-settings](#update-settings) | Master password / public access |
| [update-config-group](#update-config-group) | Attach or detach a config group |
| [update-secrule](#update-secrule) | Replace the security rules |
| [reboot](#reboot) | Restart the database |
| [delete](#delete) | Delete the cluster |

---

## list

List the clusters in your project.

### Synopsis

```
grn vdb postgresql cluster list
    [--page <value>]
    [--page-size <value>]
    [--name <value>]
    [--status <value>]
```

### Options

`--page` (integer)
: Page number, 1-based. Default: `1`.

`--page-size` (integer)
: Number of items per page. Default: `50`.

`--name` (string)
: Filter by cluster name, substring match, case-insensitive.

`--status` (string)
: Filter by status; comma-separated for several, e.g. `--status ACTIVE,BUILDING`.

!!! warning "Filtering happens on the client here"
    This calls the Relational Database listing — the only endpoint that enumerates
    clusters — and keeps the `pg-` rows. That endpoint's server-side `name` and
    `status` filters **do not apply to cluster rows**, so the CLI applies them
    itself. Consequences: `--page`/`--page-size` page the underlying mixed
    listing, so a page can return fewer clusters than `--page-size`, and
    `pageObject` in JSON output describes that mixed listing, not the filtered
    result.

### Output

`--output table` shows `id`, `name`, `status`, `datastoreVersion`,
`numberOfNodes`, `vcpus`, `ram`, `volumeSize`, `privateRwIp`, `created`.

```bash
grn vdb postgresql cluster list --output table
grn vdb postgresql cluster list --status ACTIVE --query 'data[].{id:id,name:name}'
```

---

## get

Show one cluster in full. Calls the Relational Database get-by-id endpoint.

```
grn vdb postgresql cluster get --cluster-id <value>
```

Output is a key/value view of the whole object, never a fixed column set — a
cluster carries several nested arrays. Cluster-specific fields worth knowing:
`deployType` (`cluster`), `numberOfNodes`, and `privateRwIp` / `publicRwIp` /
`privateRoIp` / `publicRoIp`, which replace a single instance's `ip` list — `Rw`
is the read-write endpoint, `Ro` the read-only one. A [Multi-AZ](#multi-az)
cluster additionally carries `multiZoneInfos[]`, one entry per zone with that
zone's subnet and endpoints.

```bash
grn vdb postgresql cluster get --cluster-id pg-2e6f2253-9032-466f-975f-d6d6b6ec8330 \
    --query '{rw:privateRwIp,ro:privateRoIp,port:port,nodes:numberOfNodes}'
```

---

## list-histories

List the recorded actions of a cluster — create, resize, restart, backup — with
status and, for failures, the error message. Calls the Relational Database
histories endpoint.

```
grn vdb postgresql cluster list-histories
    --cluster-id <value>
    [--page <value>]
    [--page-size <value>]
```

Table columns: `id`, `action`, `description`, `status`, `createdTime`, `errorMessage`.
`description` is what distinguishes the rows — a resize, a settings change and a
config-group attach are all recorded as `Update` — so it sits right after `action`.
`updatedTime` is JSON-only: it repeats `createdTime` on every record observed. Items are
keyed under `content`, not `data`.

```bash
grn vdb postgresql cluster list-histories --cluster-id pg-... --output table
```

---

## get-volume-used

Show how much of the provisioned storage is in use. One of the cluster's own
endpoints.

```
grn vdb postgresql cluster get-volume-used --cluster-id <value>
```

The API answers with an array of human-readable sizes, one per node (e.g.
`["194M"]`), so the output is printed as-is.

---

## list-secrules

List the security group rules controlling access to a cluster. Calls the
Relational Database secrules endpoint.

```
grn vdb postgresql cluster list-secrules --cluster-id <value>
```

Table output shows `id`, `direction`, `protocol`, `portRangeMin`, `portRangeMax`,
`remoteIpPrefix`, `createdAt`. Run this before
[update-secrule](#update-secrule) — that command replaces the whole set.

---

## create

Create a cluster. One of the cluster's own endpoints.

!!! danger "This places a paid order"
    Creation goes through the order/payment flow: the API answers with an order
    (`orderId`, `orderUrl`) and builds the cluster asynchronously. A successful
    response means the order was placed, not that the cluster is ready — poll
    `cluster get`. Use `--dry-run` first; the command also confirms before
    ordering unless `--force` is given. `--user-type IAM_USER` switches from
    Checkout to Auto Payment.

### Synopsis

```
grn vdb postgresql cluster create
    --name <value>
    --datastore-version <value>
    --package-id <value>
    --volume-type-id <value>
    --volume-size <value>
    --zone-id <value>
    --subnet-ids <value>
    --username <value>
    --database-name <value>
    [--password <value>]
    [--number-of-nodes <value>]
    [--config-id <value>]
    [--public-access]
    [--backup-location-id <value>]
    [--backup-policy-id <value>]
    [--backup-point-id <value>]
    [--poc]
    [--dry-run] [--force]
```

### Options

`--name` (string) — **required**
: Cluster name.

`--datastore-version` (string) — **required**
: PostgreSQL version, e.g. `17`. See
[`catalog list-datastores`](postgresql-catalog.md#list-datastores).

`--package-id` (string) — **required**
: Flavor ID, `pgp-...`. See
[`catalog list-flavors`](postgresql-catalog.md#list-flavors).

`--volume-type-id` (string) — **required**
: Volume type ID, `pgst-...`. See
[`catalog list-volume-types`](postgresql-catalog.md#list-volume-types).

`--volume-size` (integer) — **required**
: Volume size in GB, within the type's `minVolumeSize`/`maxVolumeSize`.

`--zone-id` (string) — **required**
: Availability zone, e.g. `HCM03-1A`.

`--subnet-ids` (string) — **required**
: Subnet ID(s), comma-separated. One subnet places every node in that subnet's
zone. Several — one per zone — spread the nodes across those zones
(**Multi-AZ**); the subnets come from
[`grn vdb relational catalog list-subnets`](relational-catalog.md#list-subnets).

`--username` (string) — **required**
: Master username.

`--database-name` (string) — **required**
: Initial database. The API accepts exactly one at creation.

`--password` (string)
: Master password. Read from `$GRN_VDB_MASTER_PASSWORD` when omitted, which keeps
it out of your shell history; it is masked as `***` in `--dry-run` output.

`--number-of-nodes` (integer)
: 2–10. Default: `3`.

`--config-id` (string)
: Config group with deploy type `cluster`. See
[`catalog list-config-groups`](postgresql-catalog.md#list-config-groups).

`--public-access`
: Allow public access.

`--backup-location-id`, `--backup-policy-id` (string)
: Where and how backups are kept. See
[`catalog list-backup-locations`](postgresql-catalog.md#list-backup-locations)
and [`list-backup-policies`](postgresql-catalog.md#list-backup-policies).

`--backup-point-id` (string)
: Restore point to build the cluster from, `bk-db-pt-...`. See
[`backup list-restore-points`](postgresql-backup.md#list-restore-points).

`--poc`
: Pay with PoC credit (Auto Payment only).

!!! note "Cluster IDs are not relational IDs"
    A `pgp-`/`pgst-` value comes only from the PostgreSQL catalog. Flavors and
    volume types listed by `grn vdb relational catalog` belong to the
    single-instance product and are rejected here.

### Multi-AZ

Pass one subnet per zone in `--subnet-ids` and the nodes are spread across those
zones. Two rules are enforced before the order is placed:

- Each subnet appears once — one subnet per zone, so a repeat names a zone twice.
- At most one subnet per node — every zone you name has to receive at least one
  node.

A Multi-AZ cluster can only use the flavors and volume types that
[`catalog list-flavors --multi-zone`](postgresql-catalog.md#list-flavors) and
[`catalog list-volume-types --multi-zone`](postgresql-catalog.md#list-volume-types)
list — check those before ordering. Once built, the per-zone placement of a
Multi-AZ cluster is visible in `cluster get` as `multiZoneInfos[]`: one entry per
zone with that zone's `subnetId`, read-write / read-only endpoints and `status`.

### Examples

Single zone — every node in one subnet:

```bash
export GRN_VDB_MASTER_PASSWORD='...'

grn vdb postgresql cluster create --dry-run \
    --name analytics \
    --datastore-version 17 \
    --package-id pgp-cd1958e4-a6a6-445b-b96f-07d26b811293 \
    --volume-type-id pgst-63e28e83-165c-4a6d-8cd4-36ed37f1b65d \
    --volume-size 40 --number-of-nodes 3 \
    --zone-id HCM03-1A \
    --subnet-ids sub-66a5327f-1970-427d-b1a3-8eb146e94bab \
    --username pgadmin --database-name appdb
```

Multi-AZ — one subnet per zone, nodes spread across both:

```bash
grn vdb postgresql cluster create --dry-run \
    --name ha-analytics \
    --datastore-version 17 \
    --package-id pgp-... \
    --volume-type-id pgst-... \
    --volume-size 40 --number-of-nodes 3 \
    --zone-id HCM03-1A \
    --subnet-ids sub-...-HCM03-1A,sub-...-HCM03-1B \
    --username pgadmin --database-name appdb
```

---

## resize

Change storage size, storage type or node count. One of the cluster's own
endpoints.

!!! danger "This places a paid order"
    Same order/payment flow as create, and asynchronous. Use `--dry-run` first.

### Synopsis

```
grn vdb postgresql cluster resize
    --cluster-id <value>
    --type VOLUME-SIZE|VOLUME-TYPE|NUMBER-OF-NODES
    [--volume-size <value>]
    [--volume-type-id <value>]
    [--number-of-nodes <value>]
    [--poc]
    [--dry-run] [--force]
```

### Options

`--type` (string) — **required**
: What to change. One resize does one thing, and only that dimension's flag is
read:

| `--type` | reads | 
|---|---|
| `VOLUME-SIZE` | `--volume-size` |
| `VOLUME-TYPE` | `--volume-type-id` |
| `NUMBER-OF-NODES` | `--number-of-nodes` (2–10) |

The command refuses a `--type` whose flag is missing, rather than sending an
order that changes nothing. Storage can normally only grow.

```bash
grn vdb postgresql cluster resize --cluster-id pg-... --type VOLUME-SIZE --volume-size 80 --dry-run
grn vdb postgresql cluster resize --cluster-id pg-... --type NUMBER-OF-NODES --number-of-nodes 5
```

---

## update-settings

Change the master password, the public-access setting, or both. One of the
cluster's own endpoints.

```
grn vdb postgresql cluster update-settings
    --cluster-id <value>
    [--password <value>]
    [--public-access]
    [--dry-run] [--force]
```

Only the settings you pass are sent. This matters for `--public-access`: its zero
value is a real setting, so a body that always carried `publicAccess: false` would
cut off public connectivity every time someone rotated a password. Pass the flag
only when you mean to change exposure.

`--password` falls back to `$GRN_VDB_MASTER_PASSWORD` and is masked in `--dry-run`
output. Both changes affect live connections, so the command confirms first.

```bash
grn vdb postgresql cluster update-settings --cluster-id pg-... --public-access=false
GRN_VDB_MASTER_PASSWORD='...' grn vdb postgresql cluster update-settings --cluster-id pg-...
```

---

## update-config-group

Attach a config group to the cluster, or detach the current one. One of the
cluster's own endpoints.

```
grn vdb postgresql cluster update-config-group
    --cluster-id <value>
    (--config-id <value> | --detach)
    [--dry-run] [--force]
```

Only config groups with deploy type `cluster` are accepted. The API detaches when
it receives an empty `configGroupId`; the CLI requires the explicit `--detach`
flag instead, because an unset shell variable should not silently detach your
configuration. `--config-id` and `--detach` are mutually exclusive.

Applying configuration can restart the database, so the command confirms first.

---

## update-secrule

Set which networks may reach the cluster. Calls the Relational Database secrules
endpoint, whose body is a JSON array.

### Synopsis

```
grn vdb postgresql cluster update-secrule
    --cluster-id <value>
    --rule <key=value,...> [--rule ...]
    [--add]
    [--dry-run] [--force]
```

!!! warning "This replaces the entire rule set by default"
    The API takes the full list, so any network currently allowed and not repeated
    is revoked. Pass `--add` to keep the existing rules and append instead. The
    command prints the current set and the resulting set before doing anything;
    run it with `--dry-run` first.

### Rule syntax

Each `--rule` is a comma-separated list of `key=value` pairs, and the flag repeats:

| Key | Meaning |
|-----|---------|
| `cidr` | The allowed network, e.g. `cidr=203.0.113.0/24` (**required**) |
| `port` | Single port; defaults to `5432` |
| `port-min`, `port-max` | Port range, given together |
| `id` | Existing rule ID, to keep a rule you are re-sending |

An unknown key is an error, not an ignored field: a typo'd `cird=` would otherwise
send a rule with no network and revoke everything.

### Examples

Restrict access to one network (replacing everything else):

```bash
grn vdb postgresql cluster update-secrule --cluster-id pg-... \
    --rule cidr=203.0.113.0/24 --dry-run
```

Add an office network without touching the existing rules:

```bash
grn vdb postgresql cluster update-secrule --cluster-id pg-... --add \
    --rule 'cidr=198.51.100.0/24,port=5432'
```

---

## reboot

Restart the database on a cluster. Calls the Relational Database reboot endpoint.

```
grn vdb postgresql cluster reboot --cluster-id <value> [--dry-run] [--force]
```

Open connections are dropped, so the command confirms first.

---

## delete

Delete a cluster. Calls the Relational Database delete endpoint.

```
grn vdb postgresql cluster delete
    --cluster-id <value>
    [--create-final-backup]
    [--dry-run] [--force]
```

### Options

`--create-final-backup`
: Take one last backup before deleting.

That is the only option a cluster delete takes. The shared request schema also
carries `deleteAllBackup`, but it applies to Relational Database instances, not to
clusters — existing backups survive the deletion and remain restorable.

Before doing anything the command prints the cluster it is about to destroy (ID,
name, status, version, nodes, storage). Deleting the cluster is irreversible.

```bash
grn vdb postgresql cluster delete --cluster-id pg-... --create-final-backup --dry-run
```

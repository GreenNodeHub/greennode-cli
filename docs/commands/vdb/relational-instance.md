# relational instance

Manage vDB Relational Database (MySQL, PostgreSQL, MariaDB) instances.

```bash
grn vdb relational instance <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List Relational Database instances |
| [get](#get) | Get details of one instance |
| [list-histories](#list-histories) | List an instance's action history |
| [list-secrules](#list-secrules) | List the security rules |
| [list-replicas](#list-replicas) | List an instance's read replicas |
| [create](#create) | Create an instance (**paid order**) |
| [resize-instance](#resize-instance) | Change the flavor (**paid order**) |
| [resize-storage](#resize-storage) | Change storage size or type (**paid order**) |
| [start](#start-stop-reboot) | Start a stopped instance |
| [stop](#start-stop-reboot) | Stop an instance |
| [reboot](#start-stop-reboot) | Restart the database |
| [delete](#delete) | Delete an instance |
| [create-replica](#create-replica) | Create a read replica (**paid order**) |
| [detach-replica](#detach-replica) | Promote a replica to standalone |
| [update-settings](#update-settings) | Password, public access, backup schedule |
| [update-config-group](#update-config-group) | Attach or detach a config group |
| [update-secrule](#update-secrule) | Replace the security rules |

!!! warning "`db-` only for anything that changes state"
    The listing, `get` and `list-histories` endpoints also serve PostgreSQL
    Clusters (`pg-`), and so do the relational reboot, delete and secrule
    endpoints. Every command here that changes an instance therefore **rejects a
    `pg-` ID** — clusters are managed with
    [`grn vdb postgresql cluster`](postgresql-cluster.md).

---

## list

List Relational Database instances in your project.

### Synopsis

```
grn vdb relational instance list
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
: Filter by instance name, substring match. Applied server-side.

`--status` (string)
: Filter by instance status. Comma-separated for multiple values, e.g.
`--status ACTIVE,BUILDING`. Applied server-side.
Possible values are not fixed by the API; use the status shown in `list` output.

!!! note "Filters do not apply to PostgreSQL Cluster rows"
    This endpoint returns both Relational Database instances (IDs starting with
    `db-`) and PostgreSQL Cluster records (IDs starting with `pg-`), because the
    PostgreSQL Cluster API has no listing of its own. The `--name` and `--status`
    filters are applied only to the `db-` rows; `pg-` rows are returned on every
    query regardless of the filter.

### Output

`--output table` shows a fixed column set: `id`, `name`, `status`,
`datastoreType`, `datastoreVersion`, `vcpus`, `ram`, `volumeSize`, `ip`,
`created`.

`--output json` (the default) returns the full instance object — roughly 68
fields per instance, including billing metadata such as `cost`, `period` and
`enableAutoRenew` — alongside `pageObject`, which carries `totalPages` and
`totalElements` for paging.

### Examples

List the first page:

```bash
grn vdb relational instance list
```

Show a readable table:

```bash
grn vdb relational instance list --output table
```

Filter by name and status:

```bash
grn vdb relational instance list --name prod --status ACTIVE
```

Filter by several statuses at once:

```bash
grn vdb relational instance list --status ACTIVE,ERROR
```

Page through results:

```bash
grn vdb relational instance list --page 2 --page-size 20
```

Pull out just the fields you need:

```bash
grn vdb relational instance list --query 'data[].{id:id,name:name,status:status}'
```

---

## get

Show the full details of one database instance.

### Synopsis

```
grn vdb relational instance get
    --instance-id <value>
```

### Options

`--instance-id` (string) — **required**
: Database instance ID. Shell completion suggests the IDs in your project.

!!! note "PostgreSQL Cluster IDs work here too"
    Although this is the Relational Database endpoint, it also answers for the
    PostgreSQL Cluster IDs (`pg-`) that appear in `instance list` — verified
    against the live API. A cluster answers with `deployType: cluster`,
    `numberOfNodes` and the `privateRwIp` / `publicRwIp` fields instead of a
    single instance's `ip` list.

### Output

Always a key/value view of the whole object (~68 fields), never a fixed column
set: an instance carries four nested arrays (`ip`, `securityGroup`, `replicas`,
`sharedActions`) and a column-based table cannot pick between them reliably.

### Examples

Get one instance:

```bash
grn vdb relational instance get --instance-id db-66a37ca3-e688-4a8a-9e35-a89fa69733c6
```

Just the connection details:

```bash
grn vdb relational instance get --instance-id db-66a37ca3-e688-4a8a-9e35-a89fa69733c6 \
    --query '{ip:ip,port:port,publicAccess:publicAccess}'
```

---

## list-histories

List the recorded actions of an instance — create, update, restart, backup — with
their status and, for failures, the error message.

### Synopsis

```
grn vdb relational instance list-histories
    --instance-id <value>
    [--page <value>]
    [--page-size <value>]
```

### Options

`--instance-id` (string) — **required**
: Database instance ID. PostgreSQL Cluster IDs (`pg-`) are accepted as well.

`--page` (integer)
: Page number, 1-based. Default: `1`.

`--page-size` (integer)
: Number of items per page. Default: `50`.

This endpoint takes no filters.

### Output

`--output table` shows `id`, `action`, `description`, `status`, `createdTime`,
`errorMessage`.

`description` comes second because `action` on its own does not say what happened: a
settings change, a resize, and a config-group attach or detach are all recorded as
`Update`, and only the description tells them apart —
`Update database with following changes: Change flavor from db.s-general-2x4 (2 core, 4GB
RAM) to db.s-general-4x8 (4 core, 8GB RAM)`. `Create` and `Create replica` entries embed
the whole order cart and run past 200 characters, which is what the column costs; those
are also the two rows `action` alone already identifies.

`updatedTime` is JSON-only. It equalled `createdTime` on every record observed, so in a
table it spends 31 characters repeating the column before it.

!!! note "The item key is `content`, not `data`"
    Unlike `instance list`, this listing returns its items under `content`
    alongside `pageObject`, so a `--query` written for one does not transfer to
    the other.

### Examples

Recent actions as a table:

```bash
grn vdb relational instance list-histories \
    --instance-id db-66a37ca3-e688-4a8a-9e35-a89fa69733c6 --output table
```

Read the full description of each action:

```bash
grn vdb relational instance list-histories \
    --instance-id db-66a37ca3-e688-4a8a-9e35-a89fa69733c6 \
    --query 'content[].{action:action,when:createdTime,detail:description}'
```

Find failed actions:

```bash
grn vdb relational instance list-histories \
    --instance-id db-66a37ca3-e688-4a8a-9e35-a89fa69733c6 \
    --query "content[?status!='Finished']"
```

---

## list-secrules

List the security group rules controlling access to an instance.

```
grn vdb relational instance list-secrules --instance-id <value>
```

Table columns: `id`, `direction`, `protocol`, `portRangeMin`, `portRangeMax`,
`remoteIpPrefix`, `createdAt`. Run this before
[update-secrule](#update-secrule) — that command replaces the whole set.

---

## list-replicas

List the read replicas created from an instance.

```
grn vdb relational instance list-replicas --instance-id <value>
```

The API answers with a **reduced** view of each replica — 19 fields, with no `ip`,
`port` or `securityGroup` — so table output shows `id`, `name`, `status`,
`datastoreType`, `datastoreVersion`, `vcpus`, `ram`, `volumeSize`, `created`. Run
`instance get` on a replica ID for its full details.

An instance with no replicas returns an empty list.

---

## create

Create an instance.

!!! danger "This places a paid order"
    Creation goes through the order/payment flow: the API answers with an order
    (`orderId`, `orderUrl`) and builds the instance asynchronously. A successful
    response means the order was placed, not that the database is ready — poll
    `instance get`. Use `--dry-run` first; the command also confirms before
    ordering unless `--force` is given. `--user-type IAM_USER` switches from
    Checkout to Auto Payment.

### Synopsis

```
grn vdb relational instance create
    --name <value>
    --datastore-type <value>
    --datastore-version <value>
    --package-id <value>
    --volume-type <value>
    --volume-size <value>
    --zone-id <value>
    --subnet-ids <value>
    --username <value>
    --database-name <value>
    [--password <value>]
    [--character-set <value>] [--collate <value>]
    [--config-id <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--poc]
    [--dry-run] [--force]
```

### Options

`--name` (string) — **required**
: Instance name.

`--datastore-type` (string) — **required**
: `MySQL`, `MariaDB` or `PostgreSQL`. See
[`catalog list-datastores`](relational-catalog.md#list-datastores).

`--datastore-version` (string) — **required**
: Engine version, e.g. `8.0`.

`--package-id` (string) — **required**
: The flavor, as the **numeric id** from
[`catalog list-flavors`](relational-catalog.md#list-flavors) — e.g. `211`, not the
flavor name. Shell completion offers the valid ids once `--datastore-type` and
`--datastore-version` are on the command line (and narrows them further if
`--zone-id` is too), since the flavors endpoint needs those to answer.

`--volume-type` (string) — **required**
: The volume type **NAME** — the `type` column of
[`catalog list-volume-types`](relational-catalog.md#list-volume-types). Note the
name is **zone-dependent**: the same storage is `Gen2-NVMe2-IOPS3000` in HCM03-1A
but `Gen2-NVMe2-IOPS3000-HCM03-1B` in HCM03-1B. Always read it from that command
with `--zone-id` set to the zone you are creating in.

`--volume-size` (integer) — **required**
: Volume size in GB, within that type's `minVolumeSize`/`maxVolumeSize`.

`--zone-id` (string) — **required**
: Availability zone, e.g. `HCM03-1A`.

`--subnet-ids` (string) — **required**
: Subnet ID(s), comma-separated.

`--username` (string) — **required**
: Master username.

`--database-name` (string) — **required**
: Initial database. The API accepts exactly one at creation.

`--password` (string)
: Master password. Read from `$GRN_VDB_MASTER_PASSWORD` when omitted, which keeps
it out of your shell history; it is masked as `***` in `--dry-run` output. Must be
**8–32 characters** of letters, digits and `$ ^ _ < >` — checked before the request
goes out, so a bad one names this flag instead of coming back as a 400. (MemoryStore
requires 16–128, so a password reused across products has to fall in 16–32.)

`--character-set`, `--collate` (string)
: Character set and collation of the initial database (MySQL/MariaDB), e.g.
`utf8mb4` and `utf8mb4_general_ci`.

`--config-id` (string)
: Config group to attach. See
[`catalog list-config-groups`](relational-catalog.md#list-config-groups).

`--public-access`
: Allow public access.

`--backup-auto`, `--backup-duration`, `--backup-time`
: Enable daily automatic backup. When `--backup-auto` is given, the API requires
both a retention of **2–14 days** and a time of day (`HH:MM`); the command rejects
the combination early rather than letting the API refuse it. Passing retention or
time without `--backup-auto` is also an error, since the API would ignore them.

`--poc`
: Pay with PoC credit (Auto Payment only).

!!! note "Not the same values as PostgreSQL Cluster"
    `--package-id` here is a plain number and `--volume-type` is a name. The
    cluster group uses opaque IDs instead (`pgp-…`, `pgst-…`) and the two sets are
    not interchangeable.

### Example

```bash
export GRN_VDB_MASTER_PASSWORD='...'

grn vdb relational instance create --dry-run \
    --name my-mysql \
    --datastore-type MySQL --datastore-version 8.0 \
    --package-id 211 \
    --volume-type Gen2-NVMe2-IOPS3000 --volume-size 40 \
    --zone-id HCM03-1A \
    --subnet-ids sub-66a5327f-1970-427d-b1a3-8eb146e94bab \
    --username dbadmin --database-name appdb \
    --character-set utf8mb4 --collate utf8mb4_general_ci \
    --backup-auto --backup-duration 7 --backup-time 02:00
```

---

## resize-instance

Move an instance to a different flavor (vCPU/RAM).

```
grn vdb relational instance resize-instance
    --instance-id <value>
    --package-id <value>
    [--poc]
    [--dry-run] [--force]
```

!!! danger "This places a paid order"
    Same order/payment flow as create, asynchronous, and the database restarts on
    the new flavor.

`--package-id` is the numeric flavor id from
[`catalog list-flavors`](relational-catalog.md#list-flavors) — that command needs an
engine and version, and since neither is a flag here, this is one of the few flags with
**no shell completion**: the values would have to be resolved from the instance first.
The same applies to `create-replica` and `backup restore`.
Storage is resized separately with [resize-storage](#resize-storage).

---

## resize-storage

Grow an instance's volume, or move it to a different volume type.

```
grn vdb relational instance resize-storage
    --instance-id <value>
    [--volume-size <value>]
    [--volume-type <value>]
    [--poc]
    [--dry-run] [--force]
```

!!! danger "This places a paid order"
    Asynchronous, like every resize. Storage can normally only grow — the command
    refuses a size below the current one rather than letting the API fail.

!!! note "The API wants both fields; the CLI fills in the one you omit"
    This endpoint expects the size **and** the type on every request, using the
    current value for whatever is not changing. The command therefore reads the
    instance first and supplies the missing field, so `--volume-size 80` alone does
    not silently reset the volume type. `--dry-run` shows the resolved request. It
    also refuses a request that would change nothing.

```bash
grn vdb relational instance resize-storage --instance-id db-... --volume-size 80 --dry-run
grn vdb relational instance resize-storage --instance-id db-... --volume-type Gen2-NVMe2-IOPS5000
```

---

## start, stop, reboot

```
grn vdb relational instance start   --instance-id <value> [--dry-run] [--force]
grn vdb relational instance stop    --instance-id <value> [--dry-run] [--force]
grn vdb relational instance reboot  --instance-id <value> [--dry-run] [--force]
```

`stop` keeps the instance's storage and IP; `start` brings it back. Storage is
still billed while an instance is stopped, so stopping is not a way to pause costs
entirely.

`stop` and `reboot` drop every open connection and therefore confirm first (skip
with `--force`). `start` does not prompt.

!!! note "The API calls it `shutdown`"
    The endpoint is `/shutdown`; the CLI verb is `stop`, the canonical verb across
    `grn`.

---

## delete

Delete an instance and, optionally, its backups.

```
grn vdb relational instance delete
    --instance-id <value>
    [--create-final-backup]
    [--delete-all-backup]
    [--dry-run] [--force]
```

### Options

`--create-final-backup`
: Take one last backup before deleting.

`--delete-all-backup`
: Also delete every existing backup of this instance. This is the only way to lose
the ability to restore; the confirmation prompt says so explicitly.

Before doing anything the command prints the instance it is about to destroy —
including a replica count, since replicas may have to go first. This is
irreversible.

---

## create-replica

Create a read replica of an instance.

```
grn vdb relational instance create-replica
    --instance-id <value>
    --name <value>
    [--package-id <value>] [--volume-type <value>] [--volume-size <value>]
    [--zone-id <value>] [--subnet-ids <value>] [--config-id <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--poc]
    [--dry-run] [--force]
```

!!! danger "This places a paid order"
    A replica is a full instance and costs accordingly.

**Everything defaults to the source instance.** Engine, version, flavor, volume
type and size, zone, subnet, config group, public access and backup schedule are
read from `--instance-id`, so in practice only that and `--name` are needed. Any
other flag overrides one of those; `--dry-run` shows the resolved request.

Engine and version are deliberately **not** overridable — a replica must match its
source.

```bash
grn vdb relational instance create-replica \
    --instance-id db-66a37ca3-e688-4a8a-9e35-a89fa69733c6 \
    --name docs-agent-replica --dry-run
```

---

## detach-replica

Promote a read replica to a standalone instance.

```
grn vdb relational instance detach-replica
    --instance-id <value>
    [--dry-run] [--force]
```

Pass the **replica's** ID, not the source's — run
[list-replicas](#list-replicas) on the source to find it. Replication stops and
cannot be re-established; the replica keeps its data, carries on as a normal
instance, and keeps costing what it costs.

---

## update-settings

Change the master password, public access, or the automatic-backup schedule.

```
grn vdb relational instance update-settings
    --instance-id <value>
    [--password <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--dry-run] [--force]
```

Only the settings you pass change. That matters for `--public-access`: its zero value is
a real setting, so a body that always carried `false` would cut off public connectivity
every time someone rotated a password. Pass a flag only when you mean to change it.

The **backup schedule is the exception, and always sent**: a request without `backupAuto`
is answered with HTTP 500, so the command reads the instance first and repeats its current
schedule. That also means `--backup-time` and `--backup-duration` can be changed on their
own, as long as automatic backup is on; repeating the schedule unchanged is not reported
as a change.

`--password` falls back to `$GRN_VDB_MASTER_PASSWORD`, is masked in `--dry-run` output,
and follows the same 8–32 character rule as on [create](#create). When the API rejects a
password it echoes it back in its error message; the CLI masks any secret the request
carried before printing an error.

!!! note "Asynchronous, and the response says nothing"
    The API accepts the request and applies it in the background, answering 200
    with a payload whose fields are not meaningful (its `dbInstanceId` and
    `projectId` even arrive transposed). The command therefore reports acceptance
    and points you at `instance get` to confirm the result, rather than echoing the
    payload.

```bash
grn vdb relational instance update-settings --instance-id db-... --public-access=false
grn vdb relational instance update-settings --instance-id db-... \
    --backup-auto --backup-duration 7 --backup-time 03:00
```

---

## update-config-group

Attach a config group to an instance, or detach the current one.

```
grn vdb relational instance update-config-group
    --instance-id <value>
    (--config-id <value> | --detach)
    [--dry-run] [--force]
```

The group must match the instance's engine and version — see
[`catalog list-config-groups`](relational-catalog.md#list-config-groups). The two
flags are mutually exclusive; `--detach` is required rather than accepting an empty
`--config-id`, because an unset shell variable should not silently detach your
configuration.

!!! warning "Detaching leaves the instance RESTART_REQUIRED — and updates are then ignored"
    After a detach the instance reports `RESTART_REQUIRED`. In that state this
    endpoint still answers "accepted" but **silently does not apply** further
    changes: attaching a group again does nothing until the instance is `ACTIVE`.
    Run [reboot](#start-stop-reboot) first. The command warns when it sees a
    non-`ACTIVE` status, but the API reports success either way, so the warning is
    the only signal you get.

Applying configuration can restart the database, so the command confirms first.
Like `update-settings`, it is asynchronous and reports acceptance.

---

## update-secrule

Set which networks may reach an instance.

```
grn vdb relational instance update-secrule
    --instance-id <value>
    --rule <key=value,...> [--rule ...]
    [--add]
    [--dry-run] [--force]
```

!!! warning "This replaces the entire rule set by default"
    The API takes the full list, so any network currently allowed and not repeated
    is revoked. Pass `--add` to keep the existing rules and append instead. The
    command prints the current set and the resulting set before doing anything; run
    it with `--dry-run` first.

### Rule syntax

Each `--rule` is a comma-separated list of `key=value` pairs, and the flag repeats:

| Key | Meaning |
|-----|---------|
| `cidr` | The allowed network, e.g. `cidr=203.0.113.0/24` (**required**) |
| `port` | Single port; defaults to the instance's own port |
| `port-min`, `port-max` | Port range, given together |
| `id` | Existing rule ID, to keep a rule you are re-sending |

The default port comes from the instance itself, so a rule needs no port for the
usual case — 3306 on MySQL and MariaDB, 5432 on PostgreSQL. An unknown key is an
error, not an ignored field: a typo'd `cird=` would otherwise send a rule with no
network and revoke everything.

```bash
# Restrict to one network, replacing everything else
grn vdb relational instance update-secrule --instance-id db-... \
    --rule cidr=203.0.113.0/24 --dry-run

# Add an office network without touching the existing rules
grn vdb relational instance update-secrule --instance-id db-... --add \
    --rule 'cidr=198.51.100.0/24'
```

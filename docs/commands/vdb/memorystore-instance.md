# memorystore instance

Manage vDB MemoryStore (Redis) instances.

```bash
grn vdb memorystore instance <command> [options]
```

Two product facts shape every command here:

- **No volume.** An instance's capacity is its flavor's RAM, so there are no storage
  flags and **no `resize-storage`** — growing means a larger flavor.
- **A master password, not a database user.** Access is guarded by a password on the
  instance itself. See [Redis authentication](#redis-authentication).

Instance IDs start with `db-`, the same prefix Relational Database uses. The two
products keep separate listings, so find IDs with `instance list` here — a relational
ID passed to these commands is rejected by the API, not by the CLI.

!!! note "Zones and subnets come from the relational catalog"
    MemoryStore has no zones or subnets endpoint, so use
    [`grn vdb relational catalog list-zones`](relational-catalog.md#list-zones) and
    [`list-subnets`](relational-catalog.md#list-subnets) — they are project-wide.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List instances |
| [get](#get) | Show one instance |
| [list-histories](#list-histories) | Action history |
| [list-replicas](#list-replicas) | Read replicas |
| [list-secrules](#list-secrules) | Security rules |
| [create](#create) | Create an instance (**paid order**) |
| [resize-instance](#resize-instance) | Change the flavor (**paid order**) |
| [start, stop, reboot](#start-stop-reboot) | Lifecycle |
| [delete](#delete) | Delete an instance |
| [create-replica](#create-replica) | Create a read replica (**paid order**) |
| [detach-replica](#detach-replica) | Promote a replica to standalone |
| [update-settings](#update-settings) | Password, public access, backup schedule |
| [update-config-group](#update-config-group) | Attach or detach a config group |
| [update-secrule](#update-secrule) | Replace the security rules |

---

## list

```
grn vdb memorystore instance list
    [--page <value>] [--page-size <value>]
    [--name <value>] [--status <value>]
```

Paginated, `--page` is 1-based. `--name` and `--status` filter **server-side** — and
unlike the relational listing, this one returns only MemoryStore instances, with
nothing from another product mixed in.

Table columns: `id`, `name`, `status`, `datastoreType`, `datastoreVersion`, `vcpus`,
`ram`, `ip`, `created`. There is no volume column — `ram` is the capacity.

`--status` completion suggests the statuses present in your project, read from this
listing. (MemoryStore does have a status-enumeration endpoint, but it is outdated and
deliberately unused.)

---

## get

```
grn vdb memorystore instance get --instance-id <value>
```

Key/value view of the whole object. Porting note: the path is
`/database-instances/{id}` — no `/id/` segment, which is the relational spelling.

---

## list-histories

```
grn vdb memorystore instance list-histories
    --instance-id <value> [--page <value>] [--page-size <value>]
```

Recorded actions with status and, for failures, the error message. Columns: `id`,
`action`, `description`, `status`, `createdTime`, `errorMessage`.

`description` is the field that identifies the action: `Update` covers every settings
change, resize, and config-group attach or detach, and only the description separates
them (`… Change redis password`, `… Attach config my-group`, `… Update daily backup with
time and duration from 00:00 and 2 retention to 03:30 and 4 retention`). `Create` and
`Create replica` rows carry the whole order cart there and run long.

`updatedTime` is JSON-only — it repeated `createdTime` on every record observed. Items
come back under `content`.

---

## list-replicas

```
grn vdb memorystore instance list-replicas --instance-id <value>
```

The API answers with a **reduced** view of each replica — no `ip`, no `port` — so run
`instance get` on a replica ID for its full details.

---

## list-secrules

```
grn vdb memorystore instance list-secrules --instance-id <value>
```

Run this before [update-secrule](#update-secrule), which replaces the whole set.

---

## create

Create an instance.

!!! danger "This places a paid order"
    The API answers with an order (`orderId`, `orderUrl`) and builds the instance
    asynchronously — a successful response means the order was placed, not that Redis
    is ready. Poll `instance get`. Use `--dry-run` first; `--user-type IAM_USER`
    switches from Checkout to Auto Payment.

### Synopsis

```
grn vdb memorystore instance create
    --name <value>
    --datastore-version <value>
    --package-id <value>
    --zone-id <value>
    --subnet-ids <value>
    [--datastore-type Redis]
    [--redis-password <value>] [--redis-password-enabled]
    [--config-id <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--poc]
    [--dry-run] [--force]
```

### Options

`--name`, `--datastore-version`, `--package-id`, `--zone-id`, `--subnet-ids` — **required**

`--package-id` (string)
: The flavor, as the **numeric id** from
[`catalog list-flavors`](memorystore-catalog.md#list-flavors). Shell completion offers the valid ids once
`--datastore-version` is set — and note that one flavor name can appear under two ids
in the same zone, so pick by `platformType` rather than by name.

`--datastore-type` (string)
: Defaults to `Redis`, which is what MemoryStore offers.

`--redis-password`, `--redis-password-enabled`
: See [Redis authentication](#redis-authentication). The password falls back to
`$GRN_VDB_MASTER_PASSWORD` and is masked in `--dry-run` output.

`--backup-auto`, `--backup-duration`, `--backup-time`
: Daily automatic backup. When `--backup-auto` is given the API requires both a
retention of **2–14 days** and a time of day; the command rejects the combination
early rather than letting the API refuse it.

**No storage flags**: this API takes no volume type or size.

```bash
export GRN_VDB_MASTER_PASSWORD='...'

grn vdb memorystore instance create --dry-run \
    --name my-redis --datastore-version 7.2 --package-id 284 \
    --zone-id HCM03-1B --subnet-ids sub-7cc39ad2-a00f-4edd-8e8b-5f5aaa3eeffe \
    --public-access --backup-auto --backup-duration 3 --backup-time 02:00
```

---

## resize-instance

```
grn vdb memorystore instance resize-instance
    --instance-id <value> --package-id <value> [--poc] [--dry-run] [--force]
```

!!! danger "This places a paid order"
    Asynchronous, and Redis restarts on the new flavor.

**This is the only resize MemoryStore has.** An instance holds its data in the
flavor's RAM and has no volume, so growing capacity means a larger flavor.

---

## start, stop, reboot

```
grn vdb memorystore instance start   --instance-id <value> [--dry-run] [--force]
grn vdb memorystore instance stop    --instance-id <value> [--dry-run] [--force]
grn vdb memorystore instance reboot  --instance-id <value> [--dry-run] [--force]
```

`stop` and `reboot` drop every client connection and confirm first; `start` does not
prompt.

!!! warning "Redis is in-memory"
    Stopping or rebooting loses anything the instance's own snapshot or AOF settings
    have not persisted. The confirmation says so.

The API calls the stop endpoint `/shutdown`; the CLI verb is `stop`.

---

## delete

```
grn vdb memorystore instance delete
    --instance-id <value>
    [--create-final-backup] [--delete-all-backup]
    [--dry-run] [--force]
```

Prints the instance it is about to destroy — including a replica count, since replicas
may have to go first — then confirms. `--delete-all-backup` is the only way to lose the
ability to restore.

---

## create-replica

```
grn vdb memorystore instance create-replica
    --instance-id <value> --name <value>
    [--package-id <value>] [--zone-id <value>] [--subnet-ids <value>]
    [--config-id <value>] [--redis-password <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--poc] [--dry-run] [--force]
```

!!! danger "This places a paid order"
    A replica is a full instance and costs accordingly.

**Everything defaults to the source instance**: engine, version, flavor, zone, subnet,
config group, public access and backup schedule are read from `--instance-id`, so in
practice only that and `--name` are needed. Engine and version are deliberately not
overridable — a replica must match its source.

If the source requires a master password, the replica needs one supplied: the source's
cannot be read back.

---

## detach-replica

```
grn vdb memorystore instance detach-replica --instance-id <value> [--dry-run] [--force]
```

Pass the **replica's** ID, not the source's — [list-replicas](#list-replicas) on the
source finds it. Replication stops and cannot be re-established.

---

## update-settings

```
grn vdb memorystore instance update-settings
    --instance-id <value>
    [--redis-password <value>] [--redis-password-enabled]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--dry-run] [--force]
```

Only the settings you pass change, with one exception: the **backup schedule is always
sent**. A request without `backupAuto` is answered with HTTP 500, so the command reads
the instance first and repeats its current schedule — which is why `--backup-time` and
`--backup-duration` can also be changed on their own, as long as automatic backup is on.
Repeating the schedule is not reported as a change.

!!! note "Asynchronous, and the response says nothing"
    The API accepts the request and applies it in the background, answering with a
    payload whose fields are not meaningful. The command reports acceptance and points
    you at `instance get`. It also warns when the instance is not `ACTIVE`: in that
    state these endpoints are known to answer "accepted" and silently drop the change.

!!! warning "Accepted is not applied — wait for ACTIVE between changes"
    Fire a second change while the first is still being applied and the API accepts it
    too, then drops it. The instance briefly goes `BUILDING`, and the discarded change
    appears as a `Failed` row in
    [`list-histories`](#list-histories) with `Cannot perform action EDIT, database […]
    status is BUILDING`. Both were observed live. Poll `instance get` for `ACTIVE`
    before the next call, and check `list-histories` when a change seems to have had no
    effect.

```bash
GRN_VDB_MASTER_PASSWORD='...' grn vdb memorystore instance update-settings --instance-id db-...
grn vdb memorystore instance update-settings --instance-id db-... --public-access=false

# The schedule alone, without naming --backup-auto
grn vdb memorystore instance update-settings --instance-id db-... --backup-time 03:30
```

---

## update-config-group

```
grn vdb memorystore instance update-config-group
    --instance-id <value>
    (--config-id <value> | --detach)
    [--dry-run] [--force]
```

The group must match the instance's engine and version — see
[`catalog list-config-groups`](memorystore-catalog.md#list-config-groups). The two
flags are mutually exclusive; `--detach` is required rather than accepting an empty
`--config-id`, because an unset shell variable should not silently detach your
configuration.

Detaching sends `configId: null`. The relational endpoint rejects the empty string the
spec describes, and this request shares that schema.

Applying parameters can restart Redis, so the command confirms first. Asynchronous,
like `update-settings`.

!!! note "Where the attachment shows up"
    `instance get` reports it under `configuration` (`{id, name}`) — the sibling
    `configId` field stays **null** even while a group is attached, so a script reading
    `configId` will conclude there is none. A detach leaves the instance
    `RESTART_REQUIRED`; clear it with [`reboot`](#reboot).

---

## update-secrule

```
grn vdb memorystore instance update-secrule
    --instance-id <value>
    --rule <key=value,...> [--rule ...]
    [--add]
    [--dry-run] [--force]
```

!!! warning "This replaces the entire rule set by default"
    Any network currently allowed and not repeated is revoked. Pass `--add` to keep the
    existing rules and append instead. The command prints the current set and the
    resulting set before doing anything.

Each `--rule` is a comma-separated list of `key=value` pairs: `cidr` (**required**),
`port` (defaults to the instance's own port, 6379), `port-min`/`port-max`, and `id` to
keep a rule you are re-sending. An unknown key is an error, not an ignored field.

```bash
grn vdb memorystore instance update-secrule --instance-id db-... --add \
    --rule cidr=198.51.100.0/24
```

---

## Redis authentication

Access is a master password on the instance, not a database user. The CLI enforces two
API rules so a mistake surfaces before the request goes out:

- **Public access requires the password.** `--public-access` without one is refused.
- **Giving a password means requiring it.** Passing `--redis-password` also sets
  `redisPasswordEnabled`, so the body cannot both set a password and disable it.
  Override with an explicit `--redis-password-enabled=false`.

On `update-settings` the API additionally needs an `editRedisPassword` marker alongside
any password change. That is not a setting, so the CLI sets it for you rather than
exposing a flag.

### What the password may contain

Checked before the request goes out, so a rejection names your flag instead of arriving
as a 400 with the password in it:

| Rule | MemoryStore | Relational Database |
|---|---|---|
| Length | 16–128 | 8–32 |
| Characters | letters, digits, `$ ^ _ < >` | letters, digits, `$ ^ _ < >` |
| Both `<` and `>` in one password | rejected | not enforced |

The two length ranges overlap only between **16 and 32** — worth knowing when one
`$GRN_VDB_MASTER_PASSWORD` is reused across products. The `<`/`>` pair is a live
finding: either bracket alone is accepted, but a password containing both is refused
however far apart they sit. It is enforced for MemoryStore only, the API where it was
observed.

PostgreSQL Cluster is checked by neither rule: it is a separate API surface, already
known to validate differently, and its password limits were never confirmed.

The password is read from `$GRN_VDB_MASTER_PASSWORD` when the flag is omitted, which
keeps it out of your shell history, and is masked as `***` in `--dry-run` output. When
the API rejects a password it echoes it back in the error message; the CLI masks any
secret the request carried before printing an error, so it does not reach your terminal
or your CI log.

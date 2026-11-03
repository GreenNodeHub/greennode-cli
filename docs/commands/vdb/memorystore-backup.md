# memorystore backup

Backups of vDB MemoryStore instances.

```bash
grn vdb memorystore backup <command> [options]
```

Backups consume the free allowance that comes with an instance's flavor first; past
that they need purchased quota — see [get-free-storage](#get-free-storage) and
[`backup-storage`](memorystore-backup-storage.md).

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List backups, all or per instance |
| [get](#get) | Show one backup record |
| [create](#create) | Take a backup now |
| [delete](#delete) | Delete a backup |
| [restore](#restore) | Create a **new instance** from a backup (**paid order**) |
| [get-free-storage](#get-free-storage) | Free allowance and current usage |

---

## list

```
grn vdb memorystore backup list
    [--instance-id <value>] [--page <value>] [--page-size <value>]
```

Without `--instance-id` the listing is paginated (items under `content`). With it, a
**different endpoint** — `/database-instances/{id}/backups` — returns every backup of
that instance as a plain array, and the paging flags do not apply.

The per-instance view drops the `dbInstanceId` and `instanceName` columns: that endpoint
leaves them empty, since you asked by instance.

---

## get

```
grn vdb memorystore backup get --backup-id <value>
```

The whole record as key/value. It carries the spec of the instance the backup came from
— flavor, config group, subnets — which is what [restore](#restore) reads.

!!! warning "A missing backup comes back as HTTP 200"
    This API reports "not found" as `{"code":200,"message":"success","data":null}`, not
    a 404 — a backup whose creation failed reads exactly that way. The command turns it
    into a clear error; check
    [`instance list-histories`](memorystore-instance.md#list-histories) to see why a
    backup never materialised.

Path note: `/backups/{id}/detail` here, `/backups/detail/{id}` in relational.

---

## create

```
grn vdb memorystore backup create
    --instance-id <value>
    --name <value>
    --description <value>
    [--backup-type FULL|INCREMENTAL] [--parent-id <value>]
    [--dry-run] [--force]
```

!!! danger "--description is required even though the API calls it optional"
    On the relational endpoint — which shares this request schema — a backup without a
    non-empty description is accepted (HTTP 200, `success: true`, a backup ID handed
    back) and then fails in the background with *"An error occurred when communicating
    with system"*. Measured 4 failures out of 4 there. The CLI requires it here rather
    than risk the same silent failure.

`--backup-type INCREMENTAL` builds on `--parent-id`, which is then required.

The backup counts towards billable backup storage, so the command confirms first.

---

## delete

```
grn vdb memorystore backup delete --backup-id <value> [--dry-run] [--force]
```

Prints the backup it is about to delete, then confirms. Irreversible, and deleting a
`FULL` backup that `INCREMENTAL` ones build on may leave those unrestorable.

Request shape: a **POST** to `/backups/delete` with a JSON **array** body and no ID in
the path — relational uses `DELETE` with the ID in the path instead.

---

## restore

Create a **new** instance from a backup.

```
grn vdb memorystore backup restore
    --backup-id <value>
    --name <value>
    --zone-id <value>
    --subnet-ids <value>
    [--package-id <value>] [--config-id <value>]
    [--redis-password <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--poc] [--dry-run] [--force]
```

!!! danger "This does not restore in place"
    The API builds a **new** instance and leaves the original untouched. A restore is
    therefore a create: order/payment flow, real cost, asynchronous, and it needs a
    name.

`--zone-id` and `--subnet-ids` are **required**: a restored instance need not sit where
the original did, so the CLI does not inherit them. Everything else defaults to the
backup — engine, version, flavor, config group.

!!! warning "Restoring into a different zone needs a zone-matched flavor"
    Flavor ids are zone-specific. When the target zone differs from the original's, pass
    `--package-id` from that zone's [catalog](memorystore-catalog.md) as well.
    On the relational endpoint, omitting the zone produced an error blaming the flavor
    instead of the zone — sending it explicitly is what avoids that class of confusion.

`--public-access` requires a master password, as it does on create.

---

## get-free-storage

```
grn vdb memorystore backup get-free-storage
```

Shows `freeBackupStorage` and `backupUsage`.

!!! note "The allowance is not a fixed project quota"
    It is the **sum of your instances' own allowances**, so it grows and shrinks as
    instances come and go (verified live on the relational side). Do not treat a
    comfortable margin as permanent if you are about to delete instances.

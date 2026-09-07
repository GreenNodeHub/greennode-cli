# relational backup

Backups of vDB Relational Database instances.

```bash
grn vdb relational backup <command> [options]
```

Backups consume the free allowance that comes with an instance's flavor first; past
that they need purchased quota — see [get-free-storage](#get-free-storage) and
[`backup-storage`](relational-backup-storage.md).

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
grn vdb relational backup list [--instance-id <value>] [--page <value>] [--page-size <value>]
```

Without `--instance-id` the listing is paginated (items under `content`). With it,
a **different endpoint** returns every backup of that instance as a plain array and
the paging flags do not apply.

The per-instance view also drops two columns: that endpoint leaves `dbInstanceId`
and `instanceName` empty on every row, since you asked by instance.

```bash
grn vdb relational backup list --output table
grn vdb relational backup list --instance-id db-... --output table
```

---

## get

```
grn vdb relational backup get --backup-id <value>
```

Prints the whole record as key/value. Beyond the obvious fields it carries the spec
of the instance the backup came from — flavor, storage type and size, config group,
username — which is what [restore](#restore) reads.

!!! warning "A missing backup comes back as HTTP 200"
    This API reports "not found" as `{"code":200,"message":"success","data":null}`,
    not a 404 — a backup whose creation failed reads exactly that way. The command
    turns it into a clear error; check
    [`instance list-histories`](relational-instance.md#list-histories) to see why a
    backup never materialised.

---

## create

```
grn vdb relational backup create
    --instance-id <value>
    --name <value>
    --description <value>
    [--backup-type FULL|INCREMENTAL] [--parent-id <value>]
    [--dry-run] [--force]
```

### Options

`--instance-id`, `--name` — **required**

`--description` (string) — **required**
: Free-text description.

!!! danger "--description is required even though the API calls it optional"
    Without a **non-empty** description the request is accepted — HTTP 200,
    `success: true`, a backup ID handed back — and the backup then fails in the
    background with *"An error occurred when communicating with system"*. Measured
    4 failures out of 4 without one (including an explicit empty string), against
    2 successes out of 2 with one. The CLI requires it so the failure surfaces
    before anything is sent.

`--backup-type` (string)
: `FULL` (default) or `INCREMENTAL`. Incremental builds on `--parent-id`, which is
then required.

The backup counts towards billable backup storage, so the command confirms first.

```bash
grn vdb relational backup create --instance-id db-... \
    --name nightly-manual --description "before schema change"
```

---

## delete

```
grn vdb relational backup delete --backup-id <value> [--dry-run] [--force]
```

Prints the backup it is about to delete, then confirms. Irreversible.

Deleting a `FULL` backup that `INCREMENTAL` backups build on may leave those
unrestorable — check `backup list --instance-id <id>` for children first.

The request is one of vdb's six JSON **array** bodies, and the backup ID travels in
the path *and* in the array.

---

## restore

Create a **new** instance from a backup.

```
grn vdb relational backup restore
    --backup-id <value>
    --name <value>
    --zone-id <value>
    --subnet-ids <value>
    [--package-id <value>] [--volume-type <value>] [--volume-size <value>]
    [--config-id <value>] [--password <value>]
    [--public-access]
    [--backup-auto --backup-duration <days> --backup-time <HH:MM>]
    [--poc]
    [--dry-run] [--force]
```

!!! danger "This does not restore in place"
    The API builds a **new** instance and leaves the original untouched. A restore is
    therefore a create: order/payment flow, real cost, asynchronous, and it needs a
    name.

### Placement is your choice

`--zone-id` and `--subnet-ids` are **required**: a restored instance need not sit
where the original did, so the CLI does not inherit them. (The backup record could
not supply the subnet anyway — its `netIds` field holds the *network* the original
instance sat in, not a subnet.)

!!! warning "Restoring into a different zone needs zone-matched flavor and storage"
    Flavor ids and volume type names are zone-specific: id `180` and
    `Gen2-NVMe2-IOPS3000-HCM03-1B` exist only in HCM03-1B. When the target zone
    differs from the original's, pass `--package-id` and `--volume-type` from that
    zone's [catalog](relational-catalog.md) as well. Otherwise the API answers
    `Package ID … is invalid; Volume type … is invalid` — an error that names the
    flavor and storage while the real problem is the zone.

Everything else defaults to the backup: engine, version, flavor, storage type and
size, and config group. `--dry-run` shows the resolved request.

```bash
grn vdb relational backup restore --backup-id bk-... --name restored-db \
    --zone-id HCM03-1B --subnet-ids sub-... --dry-run
```

---

## get-free-storage

```
grn vdb relational backup get-free-storage
```

Shows `freeBackupStorage` (the allowance in GB that comes with your instances'
flavors) and `backupUsage` (what your backups occupy now).

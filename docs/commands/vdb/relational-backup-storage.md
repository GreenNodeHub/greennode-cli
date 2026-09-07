# relational backup-storage

Backup storage is a purchased quota, not a per-instance setting.

```bash
grn vdb relational backup-storage <command> [options]
```

Backups consume the free allowance that comes with an instance's flavor first
([`backup get-free-storage`](relational-backup.md#get-free-storage)); beyond that
they need storage bought here.

!!! note "The free allowance is not a fixed project quota"
    It is the **sum of the `freeBackupSize` of every instance you run** — 100 GB per
    4 vCPU/8 GB instance, for example. It therefore grows and shrinks as instances
    come and go: creating one raised the reported allowance from 300 to 400 GB and
    deleting it brought it back (verified live). Do not treat a comfortable margin
    as permanent if you are about to delete instances.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | Storage you own, with usage |
| [list-packages](#list-packages) | Packages available to buy |
| [create](#create) | Buy a package (**paid order**) |
| [resize](#resize) | Move to another package (**paid order**) |
| [delete](#delete) | Release storage |

---

## list

```
grn vdb relational backup-storage list
```

What you own: `id`, `name`, `usage`, `quota`, `status`, `backupPackageName`,
`engineGroup`. No pagination or filters. An empty result means you own none and are
living within the free allowance.

`backupPackageName` comes back null even for storage bought minutes ago; the quota
column is the reliable indicator of which package it is.

!!! note "Two similar paths"
    This reads `/backup-storages/information` — what you *have*. `/backup-storages`
    is the catalogue of what you can *buy*, which is [list-packages](#list-packages).

---

## list-packages

```
grn vdb relational backup-storage list-packages
```

Table columns: `engineGroup`, `packageId`, `packageName`, `packageQuota`, `price`,
`sku`, `description`. The `packageId` is what `--package-id` expects below.

The API nests packages under engine groups; the table flattens that to one row per
package, keeping `engineGroup` on each. `--output json` shows the original shape.

---

## create

```
grn vdb relational backup-storage create --package-id <value> [--dry-run] [--force]
```

Buys a package. Order/payment flow, asynchronous. If you already own storage,
growing it is [resize](#resize), not a second purchase.

---

## resize

```
grn vdb relational backup-storage resize
    --storage-id <value>
    --package-id <value>
    [--poc]
    [--dry-run] [--force]
```

Moves existing storage to another package. Shrinking below what your backups already
occupy is rejected by the API — check the `usage` column in [list](#list) first.

---

## delete

```
grn vdb relational backup-storage delete --storage-id <value> [--dry-run] [--force]
```

Releases the quota. Prints the storage and its current usage first.

Any backup that relies on this storage is at risk: release it only when your backups
fit in the free allowance, or after deleting them.

!!! note "Storage IDs are not instance IDs"
    They look like `db-bk-storage-…`, which starts the same way as an instance ID —
    the commands check the full prefix so an instance ID cannot slip through.

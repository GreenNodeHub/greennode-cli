# memorystore backup-storage

Backup storage is a purchased quota, not a per-instance setting.

```bash
grn vdb memorystore backup-storage <command> [options]
```

Backups consume the free allowance that comes with an instance's flavor first
([`backup get-free-storage`](memorystore-backup.md#get-free-storage)); beyond that they
need storage bought here.

!!! danger "Two paths mean the opposite of their relational namesakes"
    `GET /backup-storages` lists the quota you **own** here, and the packages you can
    **buy** in the Relational Database API. The catalogue here is
    [list-packages](#list-packages) (`/backup-storages/packages`), and releasing storage
    is `/actions/delete`, singular, where relational uses `/actions/deletions`. A script
    ported between the two products without checking will read the wrong thing.

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
grn vdb memorystore backup-storage list
```

What you own: `id`, `name`, `usage`, `quota`, `status`, `backupPackageName`,
`engineGroup`. No pagination or filters. An empty result means you own none and are
living within the free allowance.

`backupPackageName` comes back null even for storage bought minutes ago (observed on the
relational side); the `quota` column is the reliable indicator of which package it is.

---

## list-packages

```
grn vdb memorystore backup-storage list-packages
```

Columns: `engineGroup`, `packageId`, `packageName`, `packageQuota`, `price`, `sku`,
`description`. The `packageId` is what `--package-id` expects below, and shell completion
for that flag reads this listing.

The API nests packages under engine groups; the table flattens that to one row per
package, keeping `engineGroup` on each. `--output json` shows the original shape.

---

## create

```
grn vdb memorystore backup-storage create --package-id <value> [--dry-run] [--force]
```

Buys a package. Order/payment flow. If you already own storage, growing it is
[resize](#resize), not a second purchase.

---

## resize

```
grn vdb memorystore backup-storage resize
    --storage-id <value> --package-id <value> [--poc] [--dry-run] [--force]
```

Moves existing storage to another package. Shrinking below what your backups already
occupy is rejected by the API — check the `usage` column in [list](#list) first.

---

## delete

```
grn vdb memorystore backup-storage delete --storage-id <value> [--dry-run] [--force]
```

Releases the quota, printing the storage and its current usage first. Any backup that
relies on it is at risk: release it only when your backups fit in the free allowance, or
after deleting them.

!!! note "Storage IDs are not instance IDs"
    They look like `db-bk-storage-…`, which starts the same way as an instance ID — the
    commands check the full prefix so an instance ID cannot slip through.

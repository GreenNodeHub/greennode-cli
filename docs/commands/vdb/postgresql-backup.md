# postgresql backup

Backups of vDB PostgreSQL Clusters.

```bash
grn vdb postgresql backup <command> [options]
```

A cluster has **one backup record** — its policy, location and total size —
holding **many restore points**, the individual snapshots. Scheduled backups come
from the policy chosen at creation; `backup create` takes one immediately.

!!! warning "Every ID here is a cluster ID"
    These endpoints key backups by the cluster they belong to:
    `--cluster-id`, never a backup ID. The relational equivalent
    (`/backups/detail/{backupId}`) means the opposite, so do not carry that habit
    over. These commands also reject `db-` IDs — backups of Relational Database
    instances are a different API.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | Backup record of every cluster |
| [get](#get) | Backup record of one cluster |
| [list-restore-points](#list-restore-points) | Snapshots of one cluster |
| [create](#create) | Take a backup now |

---

## list

```
grn vdb postgresql backup list
```

The backup record of each cluster in the project. This endpoint takes no
pagination and no filters — it returns the whole list.

Table columns: `id`, `name`, `databaseId` (the cluster), `status`,
`backupEnabled`, `totalBackupSize`, `latestRecord`, `createdAt`.

`backupPolicyName` and `backupDestinationName` are not shown: the API leaves both
null and puts the real values in the nested `policy` and `backupDestination`
objects, which table output cannot reach into.

```bash
grn vdb postgresql backup list --output table
grn vdb postgresql backup list --query '[].{cluster:databaseId,policy:policy.name,size:totalBackupSize}'
```

---

## get

```
grn vdb postgresql backup get --cluster-id <value>
```

One cluster's backup record: whether backup is enabled, which policy and location
it uses, the latest record and the total size. Printed as key/value, because the
record nests the full `policy` and `backupDestination` objects.

```bash
grn vdb postgresql backup get --cluster-id pg-... \
    --query '{enabled:backupEnabled,latest:latestRecord,vault:backupDestination.name}'
```

---

## list-restore-points

```
grn vdb postgresql backup list-restore-points --cluster-id <value>
```

The individual snapshots taken for a cluster. Table columns: `id`, `backupName`,
`status`, `time`, `engineVersion`, `compressedSize`, `uncompressedSize`,
`createdAt`.

A restore point ID (`bk-db-pt-...`) is what `--backup-point-id` expects on
[`cluster create`](postgresql-cluster.md#create) when building a cluster from a
backup.

---

## create

```
grn vdb postgresql backup create --cluster-id <value> [--dry-run] [--force]
```

Take a backup now, in addition to whatever the policy schedules. The endpoint
takes no parameters — location, policy and retention come from the cluster's
backup record.

The backup counts towards the cluster's backup storage, which is chargeable beyond
the free allowance that comes with its flavor (`backupSize` in
[`catalog list-flavors`](postgresql-catalog.md#list-flavors)), so the command
confirms first unless `--force` is given.

The API answers with a plain string inside the response envelope rather than an
object.

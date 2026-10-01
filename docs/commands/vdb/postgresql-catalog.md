# postgresql catalog

Read-only lookups of what a vDB PostgreSQL Cluster can be built from.

```bash
grn vdb postgresql catalog <command> [options]
```

These are also the endpoints behind shell completion for `--datastore-version`,
`--package-id`, `--volume-type-id`, `--config-id`, `--backup-location-id` and
`--backup-policy-id` on
[`postgresql cluster create`](postgresql-cluster.md#create).

!!! warning "Cluster values are not relational values"
    Flavor IDs here start with `pgp-` and volume type IDs with `pgst-`; the spec is
    explicit that each is only compatible with a PostgreSQL Cluster. The
    similar-looking lists under
    [`grn vdb relational catalog`](relational-catalog.md) belong to the
    single-instance product and are rejected by cluster create.

## Commands

| Command | Description |
|---------|-------------|
| [list-datastores](#list-datastores) | PostgreSQL versions a cluster can run |
| [list-flavors](#list-flavors) | vCPU/RAM packages |
| [list-volume-types](#list-volume-types) | Storage types and size limits |
| [list-config-groups](#list-config-groups) | Config groups a cluster can use |
| [list-backup-locations](#list-backup-locations) | Where backups are stored |
| [list-backup-policies](#list-backup-policies) | Backup schedules and retention |

---

## list-datastores

```
grn vdb postgresql catalog list-datastores
```

Table columns: `type`, `version`, `name`, `versionName`, `licenseName`. The
`version` value is what `--datastore-version` expects.

---

## list-flavors

```
grn vdb postgresql catalog list-flavors [--zone-id <value>] [--multi-zone]
```

Table columns: `id`, `name`, `vcpus`, `ram`, `platformType`, `backupSize`,
`status`, `locateZoneId`. The `id` (`pgp-...`) is what `--package-id` expects on
create, and `backupSize` is the free backup allowance that comes with the flavor.

`--multi-zone` restricts the list to the flavors a Multi-AZ cluster can use — a
create that passes several `--subnet-ids`, one per zone (see
[Multi-AZ](postgresql-cluster.md#multi-az)).

```bash
grn vdb postgresql catalog list-flavors --zone-id HCM03-1A --output table
grn vdb postgresql catalog list-flavors --multi-zone --query '[].id'
```

---

## list-volume-types

```
grn vdb postgresql catalog list-volume-types [--zone-id <value>] [--multi-zone]
```

Table columns: `id`, `type`, `name`, `minVolumeSize`, `maxVolumeSize`, `iops`,
`status`, `zoneId`. The `id` (`pgst-...`) is what `--volume-type-id` expects;
`minVolumeSize`/`maxVolumeSize` bound `--volume-size`.

`--multi-zone` restricts the list to the volume types a Multi-AZ cluster can use.

Note that cluster storage limits differ from the single-instance ones — the
relational types allow far larger volumes.

---

## list-config-groups

```
grn vdb postgresql catalog list-config-groups
    [--page <value>]
    [--page-size <value>]
    [--all]
```

Config groups a cluster can be attached to. The cluster product has no
config-group endpoint of its own, so this reads the Relational Database config
groups and keeps only those with `deployType: cluster` — the only ones
`--config-id` accepts. Pass `--all` to see every config group, including the
single-instance ones.

Table columns: `id`, `name`, `datastoreName`, `datastoreVersionName`,
`deployType`, `created` — `instanceCount` is omitted because the API always answers 0
for it. Items are keyed under `content`, so a query goes through that key:
`--query 'content[].id'`.

---

## list-backup-locations

```
grn vdb postgresql catalog list-backup-locations
```

The vBackup locations a cluster's backups can be stored in. Table columns: `id`,
`name`, `type`, `product`, `status`, `isDefault`, `numberOfBackupInstances`. The
`id` (`bk-des-...`) is what `--backup-location-id` expects on create.

---

## list-backup-policies

```
grn vdb postgresql catalog list-backup-policies
```

The vBackup policies a cluster's backups can follow. Table columns: `id`, `name`,
`product`, `isDefault`, `backupInstanceCount`, `createdAt`. The `id`
(`bk-pol-...`) is what `--backup-policy-id` expects on create.

The schedule and retention live in the nested `config` object, so use
`--output json` or a query to read them:

```bash
grn vdb postgresql catalog list-backup-policies --query '[].{name:name,config:config}'
```

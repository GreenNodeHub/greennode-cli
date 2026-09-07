# vDB commands

`grn vdb` — manage GreenNode Database (vDB) resources.

```bash
grn vdb <group> <resource> <command> [options]
```

vDB covers four database products that share a gateway but not an API, so the
command tree keeps them apart:

| Group | Product | Status |
|-------|---------|--------|
| [relational](relational-instance.md) | Relational Database (MySQL, PostgreSQL, MariaDB) | In progress |
| [postgresql](postgresql-cluster.md) | PostgreSQL Cluster (2–10 nodes) | Available |
| [memorystore](memorystore-instance.md) | MemoryStore (Redis) | Available |
| [kafka](kafka-cluster.md) | Kafka | Available |

## Commands

| Command | Description |
|---------|-------------|
| [relational instance](relational-instance.md) | Manage Relational Database instances |
| [relational catalog](relational-catalog.md) | Look up engines, flavors, zones, networks and volume types |
| [relational backup](relational-backup.md) | Manage backups and restore them into new instances |
| [relational configuration](relational-configuration.md) | Manage database config groups |
| [relational backup-storage](relational-backup-storage.md) | Buy and manage backup storage quota |
| [postgresql cluster](postgresql-cluster.md) | Manage PostgreSQL Clusters |
| [postgresql catalog](postgresql-catalog.md) | Look up cluster versions, flavors, volume types and backup options |
| [postgresql backup](postgresql-backup.md) | Manage PostgreSQL Cluster backups |
| [memorystore instance](memorystore-instance.md) | Manage MemoryStore (Redis) instances |
| [memorystore catalog](memorystore-catalog.md) | Look up Redis versions, flavors, networks and config groups |
| [memorystore backup](memorystore-backup.md) | Manage MemoryStore backups and restore them into new instances |
| [memorystore configuration](memorystore-configuration.md) | Manage Redis config groups |
| [memorystore backup-storage](memorystore-backup-storage.md) | Buy and manage backup storage quota |
| [kafka cluster](kafka-cluster.md) | Manage Kafka clusters |
| [kafka topic](kafka-topic.md) | Manage the topics of a cluster |
| [kafka user](kafka-user.md) | Manage users and their credentials |
| [kafka configuration](kafka-configuration.md) | Manage versioned Kafka config groups |
| [kafka catalog](kafka-catalog.md) | Look up broker flavors and volume types |

## Choosing between `relational` and `postgresql`

Both run PostgreSQL, but they are separate products with separate APIs:

- **`grn vdb relational`** — single instances (also MySQL and MariaDB). IDs start
  with `db-`.
- **`grn vdb postgresql`** — clusters of 2–10 nodes, with their own flavors, volume
  types and backup service. IDs start with `pg-`.

`grn vdb relational instance list` returns **both**, because the cluster product
has no listing of its own; `grn vdb postgresql cluster list` shows only the
clusters. Values are not interchangeable between the two: a cluster flavor
(`pgp-...`) is rejected by a relational create and vice versa.

## Region availability

vDB is currently available in **HCM-3 only**. Running a `grn vdb` command with
`--region HAN` fails with an endpoint-not-found error naming the service and
region.

## Global options

`--user-type` (string)
: Billing flow used by chargeable operations (create, resize, restore).
Possible values: `ROOT_USER` (Checkout flow), `IAM_USER` (Auto Payment flow).
When omitted, the API applies its own default of `ROOT_USER`.

All [global options](../../usage/global-options.md) apply as well, including
`--output`, `--query`, `--profile` and `--region`.

## MemoryStore vs Relational Database

MemoryStore has the same five nouns as Relational Database, and two product differences
that change what the commands offer:

- **No volume.** A MemoryStore instance's capacity is its flavor's RAM, so there are no
  storage flags and no `resize-storage` — growing means a larger flavor.
- **A master password, not a database user.** Access is guarded by
  `redisPasswordEnabled` + `redisPassword` on the instance, and public access requires
  it. See [Redis authentication](memorystore-instance.md#redis-authentication).

Zones and subnets come from the **relational** catalog: MemoryStore has no endpoint for
either, and they are project-wide.

Instance IDs use the `db-` prefix in both products, so an ID alone does not tell you
which one it belongs to — the listings do, and they are separate.

### Paths that differ

Useful when porting a script or reading the API directly. Each row is the same
operation spelled two ways:

| Operation | relational | memorystore |
|---|---|---|
| get instance | `/database-instances/id/{id}` | `/database-instances/{id}` |
| update settings | `/{id}/update/setting` | `/{id}/update-setting` |
| update config group | `/{id}/update/config-group` | `/{id}/update-config-group` |
| catalog root | `/database-instances/*` | `/database/*` |
| flavor codes | `/database-instances/flavor_zones/codes` | `/database/codes` |
| volume types | `/database-instances/volume/types` | `/database/volume-types` |
| backup detail | `/backups/detail/{id}` | `/backups/{id}/detail` |
| backup delete | `DELETE /backups/{id}/delete` | `POST /backups/delete` (no ID in the path) |
| backups of an instance | `/backups/insId/{id}` | `/database-instances/{id}/backups` |
| config group detail | `GET /configurations/id?id=` | `GET /configurations/{id}/detail` |
| config group delete | `DELETE /configurations/delete` | `POST /configurations/delete` |
| **storage you own** | `/backup-storages/information` | **`/backup-storages`** |
| **storage you can buy** | `/backup-storages` | **`/backup-storages/packages`** |
| release storage | `/actions/deletions` | `/actions/delete` |

The last three invert: `GET /backup-storages` means "what I can buy" in one product and
"what I own" in the other.

## Kafka is not shaped like the other three

`grn vdb kafka` shares the gateway and almost nothing else. Do not carry habits from
the other groups over:

- **No backups, no replicas.** Kafka is the one vDB product with no backup service, so
  deleting a cluster or a topic loses the data outright.
- **No pagination and no server-side filters** on any listing. `cluster list --name` and
  `--status` are applied by the CLI to a complete response.
- **The nouns are different**: topics, users with per-topic permissions, and config
  groups that are **versioned** — a cluster attaches to a config group *version*, and a
  group is changed by adding a version, never edited in place.
- **Security rules are per rule**, added with `create-secrule` and removed with
  `delete-secrule`, where relational and memorystore replace the whole set at once. There
  is no endpoint that lists them; `cluster list-secrules` reads them out of the cluster.
- **Identifiers are opaque strings**: `clus-`, `topic-`, `user-`, `cgroup-`,
  `cgroupver-`, and — for `--flavor-id` and `--volume-type` — `flav-` and `vtype-`.
  Relational Database's numeric flavor id and named volume type have no equivalent here.
- **Broker count, volume size and volume type are three separate chargeable
  operations** (`resize-brokers`, `resize-storage`, `update-volume-type`), not one
  resize.

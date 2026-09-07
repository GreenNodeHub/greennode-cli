# memorystore catalog

Read-only lookups of what a vDB MemoryStore instance can be built from.

```bash
grn vdb memorystore catalog <command> [options]
```

These live under `/vdb-memory/v1/database/*`, not `/database-instances/*` as in the
relational API, and they back shell completion for `--package-id` (once
`--datastore-version` is set), `--datastore-version`,
`--package-id` and `--config-id` on
[`instance create`](memorystore-instance.md#create).

!!! note "Zones and subnets are not here"
    MemoryStore has no endpoint for either. Use
    [`grn vdb relational catalog list-zones`](relational-catalog.md#list-zones) and
    [`list-subnets`](relational-catalog.md#list-subnets) — zones and subnets are
    project-wide, and the product team confirmed that is where they come from.

## Commands

| Command | Description |
|---------|-------------|
| [list-engines](#list-engines) | Engines on offer |
| [list-datastores](#list-datastores) | Deployable versions |
| [list-flavors](#list-flavors) | Flavors for one version |
| [list-families](#list-families) | Instance families |
| [list-flavor-codes](#list-flavor-codes) | CPU platform codes |
| [list-networks](#list-networks) | Networks available to instances |
| [list-subnets](#list-subnets) | Networks with their subnets |
| [list-volume-types](#list-volume-types) | Volume types (informational) |
| [list-config-groups](#list-config-groups) | Config groups an instance can use |

---

## list-engines

```
grn vdb memorystore catalog list-engines
```

Columns: `name`, `description`. In practice this is Redis.

---

## list-datastores

```
grn vdb memorystore catalog list-datastores
```

Columns: `type`, `version`, `name`, `versionName`, `licenseName`. The `version` value is
what `--datastore-version` expects.

Note `type` is lowercase here (`redis`) while an instance reports `datastoreType` in
display form (`Redis`).

---

## list-flavors

```
grn vdb memorystore catalog list-flavors
    --datastore-type <value>
    --datastore-version <value>
    [--zone-id <value>]
```

Both engine flags are required by the API; run [list-datastores](#list-datastores) for
the valid pairs. The flag is `--datastore-version`, not `--version`, which is reserved
globally.

Columns: `id`, `name`, `vcpus`, `ram`, `bandwidth`, `familyType`, `platformType`,
`locateZoneId`. The `id` is what `--package-id` expects, and **`ram` is the instance's
capacity** — a MemoryStore instance has no volume.

`volumeSize` and `monthlyCost` are omitted: the API returns 0 for both here.

```bash
grn vdb memorystore catalog list-flavors \
    --datastore-type Redis --datastore-version 7.2 --zone-id HCM03-1A --output table
```

---

## list-families

```
grn vdb memorystore catalog list-families
```

Columns: `group`, `key`, `value`, `name`, `description`. As in the relational catalog,
the response mixes real families (`group: family`) with custom zones
(`group: family_custom`).

---

## list-flavor-codes

```
grn vdb memorystore catalog list-flavor-codes
```

Columns: `key`, `value`, `familyType`, `description`. Note the path is
`/database/codes`, where relational uses `/database-instances/flavor_zones/codes`.

---

## list-networks

```
grn vdb memorystore catalog list-networks
```

Columns: `id`, `displayName`, `cidr`, `status`, `createdAt`.

---

## list-subnets

```
grn vdb memorystore catalog list-subnets [--zone-id <value>]
```

Each row is a **network**, not a subnet: columns are `uuid`, `displayName`, `status`,
`networkId`, `zoneId`, and the subnets themselves are a nested array — use
`--output json` or a `--query` to see them.

```bash
grn vdb memorystore catalog list-subnets \
    --query '[].{network:displayName,subnets:subnets[].{id:uuid,cidr:cidr,zone:zoneId}}'
```

---

## list-volume-types

```
grn vdb memorystore catalog list-volume-types [--zone-id <value>]
```

**Informational only.** The API reports volume types for MemoryStore, but a MemoryStore
instance takes no volume type or size — its capacity is the flavor's RAM — so nothing on
`instance create` consumes these.

---

## list-config-groups

```
grn vdb memorystore catalog list-config-groups
```

Columns: `id`, `name`, `datastoreName`, `datastoreVersionName`, `created`. The
`instanceCount` the API returns is not shown: it is always 0, even for a group with an
instance attached, so a column of zeros would only mislead. `configuration get` lists the
real attachments under `instances`. The `id` is what `--config-id` expects on
[`instance create`](memorystore-instance.md#create) and
[`instance update-config-group`](memorystore-instance.md#update-config-group).

A group only fits an instance with the same engine and version.
[`memorystore configuration`](memorystore-configuration.md) is where groups are
created and changed; this is the lookup used when attaching one.

# relational catalog

Read-only lookups of what a vDB Relational Database instance can be built from:
engines, datastore versions, instance families, flavors, availability zones,
networks and volume types.

```bash
grn vdb relational catalog <command> [options]
```

These are the values that creation and resize commands expect, and the same
endpoints that back shell completion for `--datastore-type`,
`--datastore-version`, `--zone-id` and volume types. The vDB API declares no
enumerations anywhere, so this catalog is the only authoritative source of valid
values.

## Commands

| Command | Description |
|---------|-------------|
| [list-engines](#list-engines) | Database engines and their licences |
| [list-datastores](#list-datastores) | Deployable engine/version pairs |
| [list-flavors](#list-flavors) | Flavors (vCPU/RAM) for one engine version |
| [list-families](#list-families) | Instance families |
| [list-flavor-codes](#list-flavor-codes) | CPU platform codes |
| [list-zones](#list-zones) | Availability zones |
| [list-networks](#list-networks) | Networks available to instances |
| [list-subnets](#list-subnets) | Networks with their subnets |
| [list-volume-types](#list-volume-types) | Volume types and size limits |
| [list-config-groups](#list-config-groups) | Config groups an instance can use |

---

## list-engines

List the database engines offered by Relational Database.

```
grn vdb relational catalog list-engines
```

`--output table` shows `name` and `description`. The `image` field comes back
empty on every engine and `engineLicenses` is a nested array; both are in
`--output json` only.

```bash
grn vdb relational catalog list-engines --output table
```

---

## list-datastores

List the engine/version pairs that can actually be deployed. This is the input
you need for [list-flavors](#list-flavors).

```
grn vdb relational catalog list-datastores
```

`--output table` shows `type`, `version`, `name`, `versionName`, `licenseName`.

Note that `type` is lowercase here (`postgresql`) while `instance list` reports
`datastoreType` in display form (`PostgreSQL`). Either form is accepted by
`--datastore-type`.

```bash
grn vdb relational catalog list-datastores --output table
```

---

## list-flavors

List the flavors (vCPU/RAM combinations) available for one engine version.

### Synopsis

```
grn vdb relational catalog list-flavors
    --datastore-type <value>
    --datastore-version <value>
    [--zone-id <value>]
```

### Options

`--datastore-type` (string) — **required**
: Engine, e.g. `postgresql` or `PostgreSQL` — the API matches both. Run
[list-datastores](#list-datastores) for the valid values.

`--datastore-version` (string) — **required**
: Engine version, e.g. `15`. The flag is *not* `--version`, which is reserved as
a global flag.

`--zone-id` (string)
: Restrict results to one availability zone. Run [list-zones](#list-zones) for
the IDs.

### Output

`--output table` shows `id`, `name`, `vcpus`, `ram`, `volumeType`, `bandwidth`,
`familyType`, `platformType`, `locateZoneId`.

The same flavor `name` is repeated once per availability zone with a different
`id`, so narrow with `--zone-id` when comparing. `volumeSize` and `monthlyCost`
are omitted from the table because the API returns `0` for both here — storage is
chosen separately when creating an instance.

### Examples

```bash
grn vdb relational catalog list-flavors \
    --datastore-type postgresql --datastore-version 15 \
    --zone-id HCM03-1A --output table
```

Just the flavor names available in a zone:

```bash
grn vdb relational catalog list-flavors \
    --datastore-type mysql --datastore-version 8.0 \
    --zone-id HCM03-1A --query '[].name'
```

---

## list-families

List the instance families that flavors belong to.

```
grn vdb relational catalog list-families
```

`--output table` shows `group`, `key`, `value`, `name`, `description`.

!!! note "Two kinds of row"
    The response mixes real families with custom zones, told apart by `group`:
    `family` rows carry `key`, `value` and a `condition.codes` list of their CPU
    platform codes, while `family_custom` rows carry only a `name` and are zones,
    not families.

```bash
grn vdb relational catalog list-families --query "[?group=='family']"
```

---

## list-flavor-codes

List the CPU platform codes that flavors are grouped by (`code-a`, `code-s`,
`code-s2`).

```
grn vdb relational catalog list-flavor-codes
```

`--output table` shows `key`, `value`, `familyType`, `description`.

---

## list-zones

List the availability zones an instance can be placed in.

```
grn vdb relational catalog list-zones
```

`--output table` shows `uuid`, `name`, `status`, `zoneType`, `isDefault`,
`description`. The `uuid` is what `--zone-id` expects elsewhere (e.g.
`HCM03-1A`).

---

## list-networks

List the networks (VPCs) an instance can be attached to.

```
grn vdb relational catalog list-networks
```

`--output table` shows `id`, `displayName`, `cidr`, `status`, `createdAt`.

---

## list-subnets

List the same networks together with their subnets.

### Synopsis

```
grn vdb relational catalog list-subnets
    [--zone-id <value>]
```

### Options

`--zone-id` (string)
: Restrict results to one availability zone.

### Output

Each row is a *network*, not a subnet: `--output table` shows `uuid`,
`displayName`, `status`, `networkId`, `zoneId`. The subnets themselves are a
nested array, so use `--output json` or a `--query` to see them.

```bash
grn vdb relational catalog list-subnets \
    --query '[].{network:displayName,subnets:subnets[].{id:uuid,cidr:cidr,zone:zoneId}}'
```

---

## list-volume-types

List the volume types available for instance storage, with their size limits and
provisioned IOPS.

### Synopsis

```
grn vdb relational catalog list-volume-types
    [--zone-id <value>]
```

### Options

`--zone-id` (string)
: Restrict results to one availability zone. Volume types are per-zone, so this
is the usual way to read this list.

### Output

`--output table` shows `type`, `description`, `minVolumeSize`, `maxVolumeSize`,
`iops`, `zoneId`. `displayName` is null on every row — `description` is the
human-readable label.

!!! note "This listing wraps its array in an object"
    Unlike every other catalog command, the payload here is `{"data": [...]}`, so
    a `--query` has to go through that key: `--query 'data[].type'`, not
    `--query '[].type'`.

```bash
grn vdb relational catalog list-volume-types --zone-id HCM03-1A --output table
```

---

## list-config-groups

List the config groups a Relational Database instance can be attached to.

```
grn vdb relational catalog list-config-groups [--all]
```

Table columns: `id`, `name`, `datastoreName`, `datastoreVersionName`, `deployType`,
`created`. The `instanceCount` the API returns is left out: it is always 0, even for a
group with an instance attached, so the column would only mislead —
`configuration get` reports the real attachments under `instances`. The `id` is what
`--config-id` expects on
[`instance create`](relational-instance.md#create) and
[`instance update-config-group`](relational-instance.md#update-config-group).

A config group only fits an instance with the **same engine and version**, so check
the `datastoreName` / `datastoreVersionName` columns before attaching one.

!!! note "Cluster groups are filtered out"
    The endpoint also returns PostgreSQL Cluster config groups (`deployType:
    cluster`, IDs prefixed `pg-cfg-`), which a single instance cannot use, so they
    are hidden by default. Pass `--all` to see them, or use
    [`grn vdb postgresql catalog list-config-groups`](postgresql-catalog.md#list-config-groups).

```bash
grn vdb relational catalog list-config-groups --output table
```

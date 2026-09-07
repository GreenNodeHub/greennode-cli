# kafka catalog

Read-only lookups for what a Kafka cluster can be built from.

```bash
grn vdb kafka catalog <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list-flavors](#list-flavors) | Broker flavors |
| [list-families](#list-families) | Instance families |
| [list-flavor-codes](#list-flavor-codes) | CPU platform codes |
| [list-volume-types](#list-volume-types) | Volume types |

Kafka's catalog is smaller than the other products'. It has **no** engine, datastore,
zone, network, subnet or config-group lookup:

- **Versions** come from shell completion on `--kafka-version` — see
  [Kafka versions](#kafka-versions) below.
- **Networks and subnets** come from
  [`grn vdb relational catalog list-networks`](relational-catalog.md) and
  `list-subnets`; they are project-wide and Kafka has no endpoint for them.
- **Config groups** have their own noun,
  [`kafka configuration`](kafka-configuration.md), because they are versioned.

None of these endpoints takes a `--zone-id`, unlike their relational and memorystore
counterparts.

---

## list-flavors

```
grn vdb kafka catalog list-flavors [--datastore-type kafka] [--datastore-version <value>]
```

Columns: `flavorId`, `id`, `name`, `vcpus`, `ram`, `priceKey`, `description`,
`locateZoneId`.

!!! important "`cluster create --flavor-id` takes `flavorId`, not `id`"
    The value is the opaque `flav-…` string. The numeric `id` is the identifier
    Relational Database's `--package-id` uses; passing it here is wrong.

`--datastore-type` filters, and `kafka` is the only value that matches anything
(`?type=bogus` returns none; `kafka` and `Kafka` both return everything — the case
inconsistency vDB has elsewhere too).

`--datastore-version` is accepted by the API and currently has **no effect**: the same
flavors come back for a real version and for a made-up one, and the rows carry no
version field to filter on client-side (verified live 2026-08-17). Completion for the
flag still offers the real versions.

---

## list-families

```
grn vdb kafka catalog list-families
```

Columns: `group`, `key`, `value`, `name`, `description`. In practice these are zone/
family records with `key` and `value` null.

---

## list-flavor-codes

```
grn vdb kafka catalog list-flavor-codes
```

The CPU platform codes flavors are grouped by (`code-a`, `code-s`, `code-s2`).
Columns: `key`, `value`, `familyType`, `description`.

---

## list-volume-types

```
grn vdb kafka catalog list-volume-types
```

Columns: `kafkaUuid`, `type`, `displayName`, `minVolumeSize`, `maxVolumeSize`, `iops`,
`zoneId`.

!!! important "`cluster create --volume-type` takes `kafkaUuid`"
    The value is the opaque `vtype-…` string, **not** the display `type`
    (`kafka.Gen2-NVMe2-IOPS3000`) and not the numeric `id`. This is the opposite of
    Relational Database, where the volume type is passed by name.

This is the one Kafka catalog endpoint whose payload is not a bare array: the envelope's
`data` is `{projectId, data: [...]}`. The CLI strips that second layer, so the output is
the volume types themselves.

---

## Kafka versions

There is no command that lists Kafka versions, because there is no user-facing endpoint
that carries them. The only source is `GET /vdb-kafka/database/configs`, which returns
the service's own settings — regexes, per-user quotas, forced broker properties — and is
deliberately not exposed as a command.

The CLI reads that endpoint for one value, `kafkaVersions`, and uses it for shell
completion of `--kafka-version` and `--datastore-version`:

```bash
grn vdb kafka cluster create --kafka-version <TAB>
3.6.0  3.6.1  3.7.0
```

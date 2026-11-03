# kafka topic

Manage the topics of a vDB Kafka cluster.

```bash
grn vdb kafka topic <command> --cluster-id <value> [options]
```

Topics live under a cluster, so **every command takes `--cluster-id`**. Topic IDs start
with `topic-`.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List the topics of a cluster |
| [get](#get) | Show one topic |
| [create](#create) | Create a topic |
| [update](#update) | Change partitions, replicas or retention |
| [delete](#delete) | Delete a topic |

---

## list

```
grn vdb kafka topic list --cluster-id <value>
```

No pagination and no filters — this is the complete list. Columns: `id`, `name`,
`partitions`, `replicas`, `retentionSeconds`, `retentionBytes`, `status`, `createdAt`.

---

## get

```
grn vdb kafka topic get --cluster-id <value> --topic-id <value>
```

`createdAt` is ISO-8601 UTC here and a localized display string in the listing, as
everywhere in the Kafka API.

---

## create

```
grn vdb kafka topic create --cluster-id <value>
    --name <value> --partitions <1-2048> --replicas <value>
    [--retention-seconds <3600-7776000>]
    [--retention-bytes <value>]
```

`--replicas` cannot exceed the cluster's broker count.

Retention is **optional and omitted when not given**, so Kafka applies its own defaults;
the topic then reads back with `retentionSeconds` and `retentionBytes` as `null`.
`--retention-bytes` accepts `-1` for unlimited.

Names must match the service's own pattern — at least 6 characters, alphanumeric plus
`- _ .`, starting and ending alphanumeric.

Returns the created topic, whose status starts at `WAITING_CREATING` and reaches
`ACTIVE` after roughly half a minute. **The cluster must be `ACTIVE`**: a create against
a busy cluster is refused with `Cluster … is not active`.

---

## update

```
grn vdb kafka topic update --cluster-id <value> --topic-id <value>
    [--partitions <value>] [--replicas <value>]
    [--retention-seconds <value>] [--retention-bytes <value>]
    [--dry-run] [--force]
```

!!! warning "The request carries the whole set"
    Flags you leave out keep their current values — the command reads the topic first
    and repeats them. Without that, omitting a field would reset it. A retention field
    that is `null` on the topic and not passed here is left **out** of the request
    rather than sent as 0, which the API would reject.

Partition and replica counts normally only go **up** in Kafka, and increasing partitions
changes which partition a key lands on, so consumers relying on key ordering are
affected.

The response carries no body; the command reports the outcome and points at
[get](#get). The topic passes through `WAITING_UPDATING` / `UPDATING` before the change
shows.

---

## delete

```
grn vdb kafka topic delete --cluster-id <value> --topic-id <value>
    [--dry-run] [--force]
```

Irreversible — Kafka has no backup service, so every record in the topic is lost.

**The topic must be `ACTIVE`.** Deleting one that is still `UPDATING` is refused with
`Topic … does not have appropriate status`; poll [get](#get) first.

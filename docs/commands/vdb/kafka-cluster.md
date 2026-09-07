# kafka cluster

Manage vDB Kafka clusters.

```bash
grn vdb kafka cluster <command> [options]
```

A cluster is 3–10 brokers sharing one flavor, one volume type and one volume size per
broker. Cluster IDs start with `clus-`.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List clusters |
| [get](#get) | Show one cluster |
| [list-histories](#list-histories) | Action history |
| [list-secrules](#list-secrules) | Security rules of a cluster |
| [create](#create) | Create a cluster (**costs money**) |
| [delete](#delete) | Delete a cluster |
| [resize-brokers](#resize-brokers) | Change the broker count (**costs money**) |
| [resize-storage](#resize-storage) | Change the volume size (**costs money**) |
| [update-volume-type](#update-volume-type) | Change the volume type (**costs money**) |
| [update-authentication](#update-authentication) | Turn mTLS / SASL on or off |
| [update-config-group](#update-config-group) | Apply a config group version |
| [update-public-access](#update-public-access) | Attach or detach public addresses |
| [create-secrule](#create-secrule) | Allow a remote IP on a port |
| [delete-secrule](#delete-secrule) | Remove one rule |

---

## list

```
grn vdb kafka cluster list [--name <value>] [--status <value>]
```

The endpoint takes **no pagination and no filters** — it returns every cluster at once,
so there is no `--page`, and `--name` (substring) and `--status` (comma-separated) are
applied by the CLI after the response. That also means the result is the complete match
set, never one page of it.

Columns: `id`, `name`, `status`, `kafkaVersion`, `kafkaBrokerCount`, `kafkaStorageType`,
`kafkaStorageSize`, `vcpus`, `ram`, `createdAt`.

!!! note "`securityGroupRules` is not in the listing"
    The listing omits it even though the schema declares it. Use
    [list-secrules](#list-secrules), which reads the cluster detail.

---

## get

```
grn vdb kafka cluster get --cluster-id <value>
```

Printed as key/value: a cluster nests four arrays (`fixedIps`, `floatingIps`,
`securityGroupRules`, `kafkaStorageUsage`).

This is the only place the security rules and the attached `configGroupVersionId` are
visible — Kafka has no separate endpoint for either.

`createdAt` here is ISO-8601 UTC (`2026-08-17T03:05:03.000+00:00`), while the **listing**
returns a localized display string (`Aug 17, 2026, 10:05:03 AM`, GMT+7). Both are
passed through unchanged; never compare a value from one against a value from the other.

---

## list-histories

```
grn vdb kafka cluster list-histories --cluster-id <value>
```

Kafka records history in its own shape: `startedAt` / `finishedAt` rather than the
`createdTime` / `updatedTime` the other vDB products use, plus `clusterId`. No
pagination.

Columns: `id`, `action`, `description`, `status`, `startedAt`, `finishedAt`,
`errorMessage`. `description` earns its width — it carries the parameters of the change
(`Name: nhontt-test, Version: 3.7.0, Flavor: …, Storage: … 20 GB, Config group: null`).

---

## list-secrules

```
grn vdb kafka cluster list-secrules --cluster-id <value>
```

**Kafka has no endpoint that lists security rules.** This command reads the cluster and
prints the `securityGroupRules` nested in it. Columns: `id`, `remoteIp`, `port`,
`status`, `createdAt`.

The IDs here are what [delete-secrule](#delete-secrule) expects.

---

## create

```
grn vdb kafka cluster create
    --name <value>
    --kafka-version <value>
    --flavor-id <value>
    --volume-type <value>
    --volume-size <GB>
    --network-id <value>
    --subnet-id <value>
    [--broker-count 3]
    [--vserver-project-id <value>]
    [--mtls-authen] [--sasl-authen]
    [--config-group-version-id <value>]
    [--encryption-volume]
    [--dry-run] [--force]
```

!!! danger "This costs money"
    An order/payment flow, completed asynchronously. Run `--dry-run` first, then poll
    [get](#get) until the status is `ACTIVE`.

**The identifiers are opaque strings, not the numeric ones Relational Database uses:**

| Flag | Value | Where from |
|---|---|---|
| `--flavor-id` | `flav-…` | the `flavorId` column of [`catalog list-flavors`](kafka-catalog.md#list-flavors) — **not** `id` |
| `--volume-type` | `vtype-…` | the `kafkaUuid` column of [`catalog list-volume-types`](kafka-catalog.md#list-volume-types) — **not** `type` |
| `--kafka-version` | `3.7.0` | shell completion; see [catalog](kafka-catalog.md#kafka-versions) |
| `--config-group-version-id` | `cgroupver-…` | a config group **version**, from [`configuration get`](kafka-configuration.md#get) |

`--broker-count` is 3–10, checked before the order is placed.

`--vserver-project-id` defaults to the profile's `project_id`, so it rarely needs
passing. `--network-id` and `--subnet-id` complete from the **relational** catalog —
Kafka has no networks endpoint and both are project-wide.

Authentication is off unless `--mtls-authen` or `--sasl-authen` is given. A cluster with
neither accepts anyone who can reach its network.

---

## delete

```
grn vdb kafka cluster delete --cluster-id <value> [--dry-run] [--force]
```

Prints what it is about to delete, then confirms. **Kafka is the one vDB product with no
backup service**, so the topics and their data go with the cluster and there is nothing
to restore from.

---

## resize-brokers

```
grn vdb kafka cluster resize-brokers --cluster-id <value> --broker-count <3-10>
    [--rebalance] [--dry-run] [--force]
```

!!! danger "This costs money"
    Order/payment flow, asynchronous.

`--rebalance` moves existing partitions onto the new broker set. Without it the brokers
change but the partitions stay where they are, so a newly added broker carries no
traffic until a topic is created or rebalanced.

A resize to the count the cluster already has is refused client-side: the API accepts
such an order and then fails it in the background with an empty change description.

The arguments travel in the **query string** (`?count=&rebalance=`), so `--dry-run`
prints the query rather than a JSON body.

---

## resize-storage

```
grn vdb kafka cluster resize-storage --cluster-id <value> --volume-size <GB>
    [--dry-run] [--force]
```

!!! danger "This costs money"
    Order/payment flow, asynchronous.

The size is **per broker**. Unlike Relational Database, where one resize carries both
size and type, Kafka splits them — use [update-volume-type](#update-volume-type) for the
type. A no-op size is refused for the same reason as above.

---

## update-volume-type

```
grn vdb kafka cluster update-volume-type --cluster-id <value> --volume-type <value>
    [--dry-run] [--force]
```

!!! danger "This costs money"
    Despite the `update-` name this is an order/payment flow like the two resizes, and
    it completes asynchronously.

Takes the `kafkaUuid` (`vtype-…`) from
[`catalog list-volume-types`](kafka-catalog.md#list-volume-types).

---

## update-authentication

```
grn vdb kafka cluster update-authentication --cluster-id <value>
    [--mtls-authen] [--sasl-authen] [--dry-run] [--force]
```

!!! warning "This can cut off every client"
    Turning a mechanism off invalidates the credentials that use it. Turning both off
    leaves the cluster accepting unauthenticated clients, and the confirmation prompt
    says so explicitly.

The endpoint takes **both** switches on every call, so a flag you leave out keeps the
cluster's current value — the command reads the cluster first to fill it in. Without
that, changing one would silently turn the other off.

---

## update-config-group

```
grn vdb kafka cluster update-config-group --cluster-id <value>
    --config-group-version-id <value> [--dry-run] [--force]
```

Takes a config group **VERSION** (`cgroupver-…`), not a group. Kafka config groups are
versioned and a cluster attaches to one specific version — see
[kafka configuration](kafka-configuration.md).

---

## update-public-access

```
grn vdb kafka cluster update-public-access --cluster-id <value> [--enable]
    [--dry-run] [--force]
```

`--enable` attaches the cluster's public addresses; omitting it (or `--enable=false`)
detaches them. Turning it on exposes the brokers to the internet, filtered only by the
security rules — check [list-secrules](#list-secrules) first.

---

## create-secrule

```
grn vdb kafka cluster create-secrule --cluster-id <value>
    --remote-ip <CIDR> --port <value> [--dry-run] [--force]
```

Rules are added and removed **one at a time** here, one endpoint each — unlike
Relational Database and MemoryStore, where a single call replaces the whole rule set.
Existing rules are untouched.

`--port` must be one of `9092`, `9094`, `9096`, `9194`, `9196`; the CLI checks it so a
typo names the flag rather than arriving as a 400.

Returns the created rule, including the ID [delete-secrule](#delete-secrule) needs.

---

## delete-secrule

```
grn vdb kafka cluster delete-secrule --cluster-id <value> --secrule-id <value>
    [--dry-run] [--force]
```

Run [list-secrules](#list-secrules) for the IDs. Clients covered only by this rule lose
access as soon as it is gone.

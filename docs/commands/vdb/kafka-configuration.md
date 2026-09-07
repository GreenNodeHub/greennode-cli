# kafka configuration

Config groups hold Kafka broker properties.

```bash
grn vdb kafka configuration <command> [options]
```

!!! info "Kafka config groups are versioned and immutable"
    A **group** (`cgroup-…`) is a named container. A **version** (`cgroupver-…`) holds
    the properties. A cluster is attached to a specific *version*, not to the group.

    So there is **no `update` command**: changing a property means
    [create-version](#create-version) followed by
    [`cluster update-config-group`](kafka-cluster.md#update-config-group). This is the
    opposite of Relational Database and MemoryStore, where a group is edited in place.

Two other differences from the other products: properties are a **list of key/value
pairs**, not a map; and Kafka has no endpoint listing the valid property names, so there
is no `list-params` and no completion for them.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List config groups |
| [get](#get) | Show one group and all its versions |
| [get-version](#get-version) | Show one version |
| [create](#create) | Create a group with its first version |
| [create-version](#create-version) | Add a version to a group |
| [delete](#delete) | Delete a group and every version |

---

## list

```
grn vdb kafka configuration list
```

No pagination. Columns: `id`, `name`, `description`, `status`, `createdAt`. Each group's
versions come back **nested in its row**, so use `--output json` to see them — or
[get](#get) for one group.

---

## get

```
grn vdb kafka configuration get --config-group-id <value>
```

Printed as key/value, since the versions are nested.

This is where the version IDs for
[`cluster update-config-group`](kafka-cluster.md#update-config-group) come from.

---

## get-version

```
grn vdb kafka configuration get-version --config-group-id <value>
    --config-group-version-id <value>
```

Shows the version's `properties` plus `associatedClusterIds` /
`associatedClusterNames` — which answer "what breaks if I change this?". Check them
before applying a different version to a cluster.

---

## create

```
grn vdb kafka configuration create --name <value>
    --property <key=value> [--property ...]
    [--description <value>]
```

Creating a group is free and touches no cluster, so there is no confirmation prompt.

```bash
grn vdb kafka configuration create --name broker-tuning \
    --property num.partitions=3 \
    --property log.retention.hours=168
```

Kafka has no endpoint listing the valid property names, so they are neither validated
nor completed here — a wrong key is rejected by the API. Note the service forces some
properties regardless of what a group says (`auto.create.topics.enable`,
`allow.everyone.if.no.acl.found`).

---

## create-version

```
grn vdb kafka configuration create-version --config-group-id <value>
    [--property <key=value> ...] [--from-version <value>]
    [--dry-run] [--force]
```

This is how a config group is "edited". Creating a version changes nothing on any
cluster by itself — apply it with
[`cluster update-config-group`](kafka-cluster.md#update-config-group).

!!! warning "A version is the whole property set"
    The properties given here do **not** merge with the previous version. Pass
    `--from-version` to start from an existing version's properties and override only
    what `--property` sets.

```bash
# Copy version N and change one property
grn vdb kafka configuration create-version --config-group-id cgroup-... \
    --from-version cgroupver-... --property num.partitions=6
```

---

## delete

```
grn vdb kafka configuration delete --config-group-id <value> [--dry-run] [--force]
```

Deletes the group and every version of it. Check [get](#get) first: a version in use by
a cluster is listed in its `associatedClusterIds`.

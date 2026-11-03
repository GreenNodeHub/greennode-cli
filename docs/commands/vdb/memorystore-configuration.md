# memorystore configuration

Config groups hold Redis parameters for vDB MemoryStore instances.

```bash
grn vdb memorystore configuration <command> [options]
```

A group is tied to one engine version and only attaches to matching instances, with
[`instance update-config-group`](memorystore-instance.md#update-config-group).

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List config groups |
| [get](#get) | Show one group and its parameters |
| [create](#create) | Create an empty group |
| [update](#update) | Set parameters (**replaces the set**) |
| [delete](#delete) | Delete a group |
| [list-params](#list-params) | Parameters Redis accepts |

---

## list

```
grn vdb memorystore configuration list [--page <value>] [--page-size <value>]
```

Paginated, items under `content`. Columns: `id`, `name`, `datastoreName`,
`datastoreVersionName`, `description`, `created` — no `deployType`, which MemoryStore
groups do not carry, and no `instanceCount`: the API answers 0 for every group, verified
live against one that had an instance attached, so [get](#get) is the only way to see
what a group is used by.

---

## get

```
grn vdb memorystore configuration get --config-id <value>
```

Printed as key/value: the parameters live in a nested `values` map.

Path note: `/configurations/{id}/detail` here, where relational takes the ID as a
**query parameter** (`/configurations/id?id=…`).

---

## create

```
grn vdb memorystore configuration create
    --name <value>
    --datastore-version <value>
    [--datastore-type Redis]
    [--description <value>]
    [--dry-run]
```

Creating a group is free and touches no instance — it holds parameters until you attach
it — so there is no confirmation prompt. Set the parameters afterwards with
[update](#update).

There is no `--deploy-type`: MemoryStore has one deployment shape, unlike the relational
equivalent which offers `single_node` and `cluster`.

---

## update

```
grn vdb memorystore configuration update
    --config-id <value>
    --set <name=value> [--set ...]
    [--merge]
    [--dry-run] [--force]
```

!!! danger "The API replaces the whole parameter set"
    Whatever the group held and you do not repeat is **dropped** — verified on the
    relational endpoint, which shares this request. Pass `--merge` to keep the existing
    parameters and only add or override the ones you name. Either way the command prints
    the parameters before and after, so the effect is visible before you confirm.

Values are sent **typed**: `1` as a number, `2.5` as a float, `true` as a boolean,
anything else as a string — the API's parameter definitions are typed. Run
[list-params](#list-params) for the accepted names and ranges.

The change is asynchronous: a read straight afterwards may still show the old
parameters, so the command points you at `get` instead of printing a stale result.

```bash
# Add one parameter, keep the rest
grn vdb memorystore configuration update --config-id cfg-... --merge --set timeout=300

# Make these the only parameters
grn vdb memorystore configuration update --config-id cfg-... --set maxmemory-policy=allkeys-lru
```

---

## delete

```
grn vdb memorystore configuration delete --config-id <value> [--dry-run] [--force]
```

A group still attached to instances cannot be deleted — detach it first with
`instance update-config-group --detach`, and read [get](#get)'s `instances` to see what
is attached (`instanceCount` is always 0 and cannot be used for this). Attempting it
anyway is refused clearly: `Some configuration groups are being attached to some
database(s)`.

Deletion is not instantly visible: for a few seconds afterwards [get](#get) still
returns the old record, then starts reporting the group as not found. The listing drops
it immediately.

The request is a **POST** to `/configurations/delete` with a JSON array body and no ID
in the path; relational uses `DELETE` for the same operation.

---

## list-params

```
grn vdb memorystore configuration list-params
    --datastore-version <value> [--datastore-type Redis]
```

The Redis parameters one version accepts, with `type`, `min`, `max`, `modifiable` and
`restartRequired`. This is the reference for `update --set`: a parameter that is not
listed, or has `modifiable: false`, is rejected.

```bash
grn vdb memorystore configuration list-params --datastore-version 7.2 --output table
```

# relational configuration

Config groups hold database parameters for vDB Relational Database instances.

```bash
grn vdb relational configuration <command> [options]
```

A group is tied to one engine and version and only attaches to matching instances,
with
[`instance update-config-group`](relational-instance.md#update-config-group).

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List config groups |
| [get](#get) | Show one group and its parameters |
| [create](#create) | Create an empty group |
| [update](#update) | Set parameters (**replaces the set**) |
| [delete](#delete) | Delete a group |
| [list-params](#list-params) | Parameters an engine accepts |

---

## list

```
grn vdb relational configuration list [--page <value>] [--page-size <value>]
```

Paginated, items under `content`. Includes PostgreSQL Cluster groups (`deployType:
cluster`);
[`catalog list-config-groups`](relational-catalog.md#list-config-groups) is the
filtered view for attaching to a single instance.

---

## get

```
grn vdb relational configuration get --config-id <value>
```

!!! note "The ID goes in a query parameter"
    This endpoint is `/configurations/id?id=…`, not `/configurations/id/<id>` —
    building the path form returns 404.

Printed as key/value: the parameters live in a nested `values` map and the attached
instances in `instances`.

---

## create

```
grn vdb relational configuration create
    --name <value>
    --datastore-type <value>
    --datastore-version <value>
    [--deploy-type single_node|cluster]
    [--description <value>]
    [--dry-run]
```

Creating a group is free and touches no instance — it holds parameters until you
attach it — so there is no confirmation prompt. Set the parameters afterwards with
[update](#update).

`--deploy-type cluster` produces a group for PostgreSQL Cluster (PostgreSQL only).

---

## update

```
grn vdb relational configuration update
    --config-id <value>
    --set <name=value> [--set ...]
    [--merge]
    [--dry-run] [--force]
```

!!! danger "The API replaces the whole parameter set"
    Whatever the group held and you do not repeat is **dropped** — verified against
    the live API, and not what the spec suggests. Pass `--merge` to keep the existing
    parameters and only add or override the ones you name. Either way the command
    prints the parameters before and after, so the effect is visible before you
    confirm.

Values are sent **typed**: `1` as a number, `2.5` as a float, `true` as a boolean,
anything else as a string — the API's own parameter definitions are typed and its
example sends `{"autocommit": 1}`. Run [list-params](#list-params) for the accepted
names, ranges and whether a parameter needs a restart.

The change is asynchronous: a read straight afterwards still shows the old
parameters, so the command points you at `get` instead of printing a stale result.

```bash
# Add one parameter, keep the rest
grn vdb relational configuration update --config-id cfg-... --merge \
    --set long_query_time=5

# Make these the only parameters
grn vdb relational configuration update --config-id cfg-... \
    --set autocommit=1 --set long_query_time=5
```

---

## delete

```
grn vdb relational configuration delete --config-id <value> [--dry-run] [--force]
```

A group still attached to instances cannot be deleted — detach it first with
`instance update-config-group --detach`, then check [get](#get)'s `instances` to see what
is attached. The `instanceCount` the API returns is always 0, verified live against a
group that had an instance attached, so it cannot be used for this.

The request is one of vdb's JSON **array** bodies, with no ID in the path.

---

## list-params

```
grn vdb relational configuration list-params
    --datastore-type <value>
    --datastore-version <value>
    [--deploy-type <value>]
```

The parameters one engine version accepts, with `type`, `min`, `max`, `modifiable`
and `restartRequired`. This is the reference for `update --set`: a parameter that is
not listed, or has `modifiable: false`, is rejected.

```bash
grn vdb relational configuration list-params \
    --datastore-type MySQL --datastore-version 8.0 --output table
```

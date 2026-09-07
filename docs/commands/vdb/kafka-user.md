# kafka user

Manage the users of a vDB Kafka cluster and their credentials.

```bash
grn vdb kafka user <command> --cluster-id <value> [options]
```

Users are a **Kafka-only concept in vDB**. MemoryStore authenticates with a single
master password set through its instance settings, and Relational Database creates
database users as part of the instance order — neither has anything like this. User IDs
start with `user-`.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List the users of a cluster |
| [get](#get) | Show one user and its permissions |
| [get-creds](#get-creds) | Show a user's credentials (**secret**) |
| [create](#create) | Create a user |
| [update](#update) | Replace a user's permissions |
| [delete](#delete) | Delete a user |
| [generate-creds](#generate-creds) | Reissue a user's credentials |

## Permissions

A user holds four kinds of permission, each as a pair: a list of topic **names** and an
"all topics" switch.

| Flags | Request fields |
|---|---|
| `--produce-topics`, `--produce-all` | `produceTopicNames`, `produceAll` |
| `--consume-topics`, `--consume-all` | `consumeTopicNames`, `consumeAll` |
| `--produce-consume-topics`, `--produce-consume-all` | `produceConsumeTopicNames`, `produceConsumeAll` |
| `--admin-topics`, `--admin-all` | `adminTopicNames`, `adminAll` |

Topics are referenced by **NAME**, not by ID — the one place in the Kafka API that does
so — and shell completion offers names accordingly. Each `--*-all` switch **overrides**
its list: the API ignores the topic names when the matching switch is true, so passing
both is not an error, just a list that does nothing.

A user can only use a mechanism the cluster itself accepts. `--sasl-authen` on a user
does nothing until [`cluster update-authentication`](kafka-cluster.md#update-authentication)
enables SASL on the cluster.

---

## list

```
grn vdb kafka user list --cluster-id <value>
```

Columns: `id`, `name`, `status`, `produceAll`, `consumeAll`, `produceConsumeAll`,
`adminAll`, `mtlsAuthen`, `saslAuthen`, `createdAt`. The four topic-name lists are
arrays and do not fit a table — use `--output json`, or [get](#get) for one user.

---

## get

```
grn vdb kafka user get --cluster-id <value> --user-id <value>
```

Printed as key/value: a user nests four arrays.

---

## get-creds

```
grn vdb kafka user get-creds --cluster-id <value> --user-id <value>
```

!!! danger "The output is secret"
    This returns the credential material itself, as a bare array of strings. Do not
    paste it into a ticket or a shared log; redirect it to a file with restrictive
    permissions if it has to be kept.

---

## create

```
grn vdb kafka user create --cluster-id <value> --name <value>
    [--produce-topics a,b] [--produce-all]
    [--consume-topics a,b] [--consume-all]
    [--produce-consume-topics a,b] [--produce-consume-all]
    [--admin-topics a,b] [--admin-all]
    [--mtls-authen] [--sasl-authen]
```

Creating a user is free and touches no data, so there is no confirmation prompt.

If neither `--mtls-authen` nor `--sasl-authen` is given the command warns: the user is
created but has no way to authenticate.

Returns the created user. Read its credentials afterwards with [get-creds](#get-creds).

---

## update

```
grn vdb kafka user update --cluster-id <value> --user-id <value>
    [permission flags] [--mtls-authen] [--sasl-authen]
    [--dry-run] [--force]
```

!!! warning "The request replaces the whole permission set"
    Flags you leave out keep their current values — the command reads the user first and
    repeats them. Without that, changing one permission would revoke the other three.
    A list you **do** pass replaces the previous one; it is not merged, so run
    [get](#get) first to see what is there.

The response carries no body; the command reports the outcome and points at
[get](#get).

---

## delete

```
grn vdb kafka user delete --cluster-id <value> --user-id <value>
    [--dry-run] [--force]
```

Anything connecting as that user stops working immediately.

---

## generate-creds

```
grn vdb kafka user generate-creds --cluster-id <value> --user-id <value>
    [--dry-run] [--force]
```

!!! warning "This invalidates the current credentials"
    Every client still using them is cut off until it is given the new ones.

The API returns no usable body, so the command reports the outcome and leaves fetching
the new values to [get-creds](#get-creds) — that way the secret is printed only when it
is asked for.

(The API operation is `regenerateUserAuthenCredential`; the CLI spells it
`generate-creds`, since `regenerate` is not one of the canonical verbs.)

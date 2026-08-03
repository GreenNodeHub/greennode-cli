# network-interface

Manage elastic network interfaces (ENIs) that can be attached to vServer instances.

```bash
grn vserver network-interface <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all network interfaces |
| [create](#create) | Create a new network interface |
| [edit](#edit) | Rename a network interface |
| [update-tags](#update-tags) | Update tags on a network interface |
| [delete](#delete) | Delete a network interface |

---

## list

List all elastic network interfaces in your project.

### Synopsis

```
grn vserver network-interface list
    [--page <value>]
    [--page-size <value>]
    [--name <value>]
```

### Options

`--page` (integer)
: Page number, 1-based. Default: `1`.

`--page-size` (integer)
: Number of items per page. Default: `50`.

`--name` (string)
: Filter by interface name (substring match).

### Examples

```bash
grn vserver network-interface list
grn vserver network-interface list --output table
```

---

## create

Create a new elastic network interface.

### Synopsis

```
grn vserver network-interface create
    --name <value>
    --zone-id <value>
    [--tag <value> ...]
```

### Options

`--name` (required)
: Network interface name.

`--zone-id` (required)
: Availability zone ID.

`--tag` (string, repeatable)
: Tag in `key=value` format. Repeat the flag to add multiple tags.

### Examples

```bash
grn vserver network-interface create \
  --name my-interface \
  --zone-id zone-abc123

grn vserver network-interface create \
  --name prod-interface \
  --zone-id zone-abc123 \
  --tag env=prod \
  --tag team=backend
```

---

## edit

Rename an existing network interface.

### Synopsis

```
grn vserver network-interface edit
    --network-interface-id <value>
    --name <value>
```

### Options

`--network-interface-id` (required)
: Network interface ID.

`--name` (required)
: New name for the interface.

### Examples

```bash
grn vserver network-interface edit \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001 \
  --name new-name
```

---

## update-tags

Update the tags on a network interface.

### Synopsis

```
grn vserver network-interface update-tags
    --network-interface-id <value>
    [--tag <value> ...]
    [--edited-tag <value> ...]
```

### Options

`--network-interface-id` (required)
: Network interface ID.

`--tag` (string, repeatable)
: New tag to add in `key=value` format.

`--edited-tag` (string, repeatable)
: Existing tag being modified, in `key=value` format.

### Examples

```bash
grn vserver network-interface update-tags \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001 \
  --tag env=staging \
  --edited-tag team=backend
```

---

## delete

Delete a network interface. Shows a confirmation prompt unless `--force` is used.

### Synopsis

```
grn vserver network-interface delete
    --network-interface-id <value>
    [--force]
```

### Options

`--network-interface-id` (required)
: Network interface ID.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
grn vserver network-interface delete \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001

grn vserver network-interface delete \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001 \
  --force
```

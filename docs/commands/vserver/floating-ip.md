# floating-ip

Manage floating IPs (public WAN IPs) that can be attached to or detached from vServer instances.

```bash
grn vserver floating-ip <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all floating IPs |
| [delete](#delete) | Delete a floating IP |

---

## list

List all floating IPs in your project.

### Synopsis

```
grn vserver floating-ip list
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
: Filter by floating IP name (substring match).

### Examples

```bash
grn vserver floating-ip list
grn vserver floating-ip list --output table
grn vserver floating-ip list --query "listData[].{id: uuid, ip: ipAddress, status: status}"
```

---

## delete

Delete a floating IP. Shows a confirmation prompt unless `--force` is used.

### Synopsis

```
grn vserver floating-ip delete
    --floating-ip-id <value>
    [--force]
```

### Options

`--floating-ip-id` (required)
: Floating IP ID.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
grn vserver floating-ip delete --floating-ip-id fip-abc12345-0000-0000-0000-000000000001
grn vserver floating-ip delete --floating-ip-id fip-abc12345-0000-0000-0000-000000000001 --force
```

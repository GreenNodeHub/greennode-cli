# dhcp

Manage DHCP option sets and associate them with VPCs.

```bash
grn vserver dhcp <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all DHCP option sets |
| [get](#get) | Get details of a DHCP option set |
| [create](#create) | Create a new DHCP option set |
| [list-vpcs](#list-vpcs) | List VPCs associated with a DHCP option set |
| [associate-vpc](#associate-vpc) | Associate or detach a VPC from a DHCP option set |
| [delete](#delete) | Delete a DHCP option set |

---

## list

List all DHCP option sets in your project.

### Synopsis

```
grn vserver dhcp list
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
: Filter by DHCP option set name (substring match).

### Examples

```bash
grn vserver dhcp list
grn vserver dhcp list --output table
```

---

## get

Get details of a DHCP option set.

### Synopsis

```
grn vserver dhcp get --dhcp-option-id <value>
```

### Options

`--dhcp-option-id` (required)
: DHCP option set ID.

### Examples

```bash
grn vserver dhcp get --dhcp-option-id dhcp-abc12345-0000-0000-0000-000000000001
```

---

## create

Create a new DHCP option set. Two default DNS servers (`10.166.12.196`, `10.166.12.197`) are always included. Use `--dns-server` to add up to 2 additional DNS servers.

### Synopsis

```
grn vserver dhcp create
    --name <value>
    [--dns-server <value> ...]
```

### Options

`--name` (required)
: DHCP option set name.

`--dns-server` (string, repeatable)
: Additional DNS server IP address. Can be specified up to 2 times.

### Examples

```bash
# Default DNS only
grn vserver dhcp create --name my-dhcp

# Custom additional DNS servers
grn vserver dhcp create \
  --name custom-dns \
  --dns-server 8.8.8.8 \
  --dns-server 8.8.4.4
```

---

## list-vpcs

List all VPCs associated with a DHCP option set.

### Synopsis

```
grn vserver dhcp list-vpcs
    --dhcp-option-id <value>
    [--page <value>]
    [--page-size <value>]
```

### Options

`--dhcp-option-id` (required)
: DHCP option set ID.

`--page` (integer)
: Page number, 1-based. Default: `1`.

`--page-size` (integer)
: Number of items per page. Default: `50`.

### Examples

```bash
grn vserver dhcp list-vpcs --dhcp-option-id dhcp-abc12345-0000-0000-0000-000000000001
```

---

## associate-vpc

Associate a VPC with a DHCP option set, or detach it using `--detach`.

### Synopsis

```
grn vserver dhcp associate-vpc
    --vpc-id <value>
    [--dhcp-option-id <value>]
    [--detach]
```

### Options

`--vpc-id` (required)
: VPC ID.

`--dhcp-option-id` (string)
: DHCP option set ID to associate the VPC with. Not required when using `--detach`.

`--detach` (boolean)
: Detach the VPC from its current DHCP option set.

### Examples

```bash
# Associate a VPC
grn vserver dhcp associate-vpc \
  --vpc-id vpc-abc12345-0000-0000-0000-000000000001 \
  --dhcp-option-id dhcp-abc12345-0000-0000-0000-000000000001

# Detach a VPC from its DHCP option set
grn vserver dhcp associate-vpc \
  --vpc-id vpc-abc12345-0000-0000-0000-000000000001 \
  --detach
```

---

## delete

Delete a DHCP option set. Shows a confirmation prompt unless `--force` is used.

### Synopsis

```
grn vserver dhcp delete
    --dhcp-option-id <value>
    [--force]
```

### Options

`--dhcp-option-id` (required)
: DHCP option set ID.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
grn vserver dhcp delete --dhcp-option-id dhcp-abc12345-0000-0000-0000-000000000001
grn vserver dhcp delete --dhcp-option-id dhcp-abc12345-0000-0000-0000-000000000001 --force
```

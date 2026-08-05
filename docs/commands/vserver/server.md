# server

Manage vServer virtual machine instances.

```bash
grn vserver server <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all vServer instances |
| [get](#get) | Get details of an instance |
| [create](#create) | Create a new instance |
| [start](#start) | Start a stopped instance |
| [stop](#stop) | Stop a running instance |
| [reboot](#reboot) | Reboot an instance |
| [resize](#resize) | Change an instance to a different flavor |
| [update-secgroup](#update-secgroup) | Update the security groups attached to an instance |
| [create-image](#create-image) | Create a user image from an instance |
| [attach-floating-ip](#attach-floating-ip) | Attach a floating IP to an instance |
| [detach-floating-ip](#detach-floating-ip) | Detach a floating IP from an instance |
| [list-interfaces](#list-interfaces) | List network interfaces attached to an instance |
| [attach-internal-interface](#attach-internal-interface) | Attach an internal network interface |
| [detach-internal-interface](#detach-internal-interface) | Detach an internal network interface |
| [attach-external-interface](#attach-external-interface) | Attach an external network interface |
| [detach-external-interface](#detach-external-interface) | Detach an external network interface |
| [tag-key](#tag-key) | List available tag keys |
| [tag-value](#tag-value) | List possible values for a tag key |
| [delete](#delete) | Delete an instance |

---

## list

List all vServer instances in your project.

### Synopsis

```
grn vserver server list
    [--page <value>]
    [--page-size <value>]
    [--no-paginate]
    [--name <value>]
```

### Options

`--page` (integer)
: Page number, 1-based. Default: `1`.

`--page-size` (integer)
: Number of items per page. Default: `50`.

`--no-paginate` (boolean)
: Disable pagination and return all results.

`--name` (string)
: Filter by server name (substring match).

### Examples

```bash
# List all servers
grn vserver server list

# Filter by name
grn vserver server list --name web

# Table output
grn vserver server list --output table

# JMESPath filter — names and statuses only
grn vserver server list --query "listData[].{name: name, status: status}"
```

---

## get

Get full details of a vServer instance.

### Synopsis

```
grn vserver server get
    --server-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

### Examples

```bash
grn vserver server get --server-id srv-abc12345-0000-0000-0000-000000000001
```

---

## create

Create a new vServer instance. Several prerequisites are needed — run the discovery commands first if you don't have the IDs yet.

### Synopsis

```
grn vserver server create
    --name <value>
    --flavor-id <value>
    --image-id <value>
    --network-id <value>
    --subnet-id <value>
    --root-disk-type-id <value>
    --zone-id <value>
    [--root-disk-size <value>]
    [--root-disk-encryption-type <value>]
    [--encryption-volume]
    [--data-disk-type-id <value>]
    [--data-disk-size <value>]
    [--data-disk-encryption-type <value>]
    [--data-disk-name <value>]
    [--attach-floating]
    [--security-group <value>]
    [--ssh-key-id <value>]
    [--user-name <value>]
    [--user-password <value>]
    [--expire-password]
    [--server-group-id <value>]
    [--host-group-id <value>]
    [--enable-backup]
    [--backup-instance-point-id <value>]
    [--snapshot-instance-point-id <value>]
    [--period <value>]
    [--is-poc]
    [--is-enable-auto-renew]
    [--os-licence]
    [--user-data <value>]
    [--user-data-base64-encoded]
    [--dry-run]
```

### Options

**Instance settings**

`--name` (required)
: Server name. Must be 5–65 characters, alphanumeric, hyphens, and underscores, starting and ending with an alphanumeric character.

`--flavor-id` (required)
: Flavor (instance type) ID. Run `grn vserver flavor list --family <family> --code <code>` to browse options.

`--image-id` (required)
: OS image ID. Run `grn vserver image list --type os` to browse options.

`--zone-id` (required)
: Availability zone ID. Omit this flag to see available zones printed to stderr.

**Networking**

`--network-id` (required)
: VPC (network) ID. Run `grn vserver vpc list` to browse options.

`--subnet-id` (required)
: Subnet ID within the VPC. Run `grn vserver subnet list --vpc-id <vpc-id>` to browse options.

`--attach-floating` (boolean)
: Attach a floating (public) IP to the server. Default: `false`.

`--security-group` (string)
: Comma-separated list of security group IDs to attach.

**Root disk**

`--root-disk-type-id` (required)
: Volume type ID for the root disk. Run `grn vserver volume-type list --zone-id <zone-id> --type SSD` to browse options.

`--root-disk-size` (integer)
: Root disk size in GiB. Minimum: `20`. Default: `20`.

`--root-disk-encryption-type` (string)
: Encryption type for the root disk.

`--encryption-volume` (boolean)
: Encrypt the root volume.

**Data disk (optional)**

`--data-disk-type-id` (string)
: Volume type ID for an optional data disk.

`--data-disk-size` (integer)
: Data disk size in GiB. Set to `0` to skip the data disk.

`--data-disk-encryption-type` (string)
: Encryption type for the data disk.

`--data-disk-name` (string)
: Name for the data disk.

**Authentication**

`--ssh-key-id` (string)
: SSH key pair ID to inject into the server.

`--user-name` (string)
: OS login username.

`--user-password` (string)
: OS login password.

`--expire-password` (boolean)
: Force a password change on first login. Default: `true`.

**Placement**

`--server-group-id` (string)
: Server group ID for placement affinity/anti-affinity policy.

`--host-group-id` (string)
: Dedicated host group ID.

**Billing**

`--period` (integer)
: Billing period in months. Default: `1`.

`--is-poc` (boolean)
: Mark as a proof-of-concept (PoC) instance.

`--is-enable-auto-renew` (boolean)
: Enable auto-renewal.

`--os-licence` (boolean)
: Include OS licence in billing.

**Backup and restore**

`--enable-backup` (boolean)
: Enable backup for the server.

`--backup-instance-point-id` (string)
: Restore from a backup instance point.

`--snapshot-instance-point-id` (string)
: Restore from a snapshot instance point.

**User data**

`--user-data` (string)
: Cloud-init user data script.

`--user-data-base64-encoded` (boolean)
: Indicate that `--user-data` is already base64-encoded.

**Other**

`--dry-run` (boolean)
: Validate all parameters and print a report without creating the server.

### Examples

```bash
# Minimal server
grn vserver server create \
  --name my-server \
  --flavor-id flv-2c4g \
  --image-id img-ubuntu-22-04 \
  --network-id net-abc12345-0000-0000-0000-000000000001 \
  --subnet-id sub-abc12345-0000-0000-0000-000000000001 \
  --root-disk-type-id vtype-ssd-123 \
  --zone-id zone-abc123

# Server with floating IP, security group, and SSH key
grn vserver server create \
  --name web-server \
  --flavor-id flv-4c8g \
  --image-id img-ubuntu-22-04 \
  --network-id net-abc12345-0000-0000-0000-000000000001 \
  --subnet-id sub-abc12345-0000-0000-0000-000000000001 \
  --root-disk-type-id vtype-ssd-123 \
  --root-disk-size 50 \
  --zone-id zone-abc123 \
  --attach-floating \
  --security-group sg-aaa111,sg-bbb222 \
  --ssh-key-id key-abc12345-0000-0000-0000-000000000001

# Dry-run to validate parameters
grn vserver server create \
  --name my-server \
  --flavor-id flv-2c4g \
  --image-id img-ubuntu-22-04 \
  --network-id net-abc12345-0000-0000-0000-000000000001 \
  --subnet-id sub-abc12345-0000-0000-0000-000000000001 \
  --root-disk-type-id vtype-ssd-123 \
  --zone-id zone-abc123 \
  --dry-run
```

---

## start

Start a stopped vServer instance.

### Synopsis

```
grn vserver server start --server-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

### Examples

```bash
grn vserver server start --server-id srv-abc12345-0000-0000-0000-000000000001
```

---

## stop

Stop a running vServer instance.

### Synopsis

```
grn vserver server stop --server-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

### Examples

```bash
grn vserver server stop --server-id srv-abc12345-0000-0000-0000-000000000001
```

---

## reboot

Reboot a vServer instance.

### Synopsis

```
grn vserver server reboot --server-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

### Examples

```bash
grn vserver server reboot --server-id srv-abc12345-0000-0000-0000-000000000001
```

---

## resize

Change a vServer instance to a different flavor (instance type). The server must be stopped before resizing.

### Synopsis

```
grn vserver server resize
    --server-id <value>
    --flavor-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--flavor-id` (required)
: New flavor ID. Run `grn vserver flavor list --family <family> --code <code>` to browse options.

### Examples

```bash
grn vserver server resize \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --flavor-id flv-8c16g
```

---

## update-secgroup

Replace the security groups attached to a vServer instance.

### Synopsis

```
grn vserver server update-secgroup
    --server-id <value>
    --security-group <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--security-group` (required)
: Comma-separated list of security group IDs. Replaces the full set of attached security groups.

### Examples

```bash
grn vserver server update-secgroup \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --security-group sg-aaa111,sg-bbb222
```

---

## create-image

Create a user image (snapshot) from a vServer instance. The resulting image appears in `grn vserver user-image list`.

### Synopsis

```
grn vserver server create-image
    --server-id <value>
    --name <value>
    [--tag <value> ...]
```

### Options

`--server-id` (required)
: Server UUID.

`--name` (required)
: Name for the new user image.

`--tag` (string, repeatable)
: Tag in `key=value` format. Repeat the flag to add multiple tags.

### Examples

```bash
grn vserver server create-image \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --name my-golden-image

grn vserver server create-image \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --name prod-snapshot \
  --tag env=prod \
  --tag version=v2
```

---

## attach-floating-ip

Attach a floating IP to a server's network interface.

### Synopsis

```
grn vserver server attach-floating-ip
    --server-id <value>
    --floating-ip-id <value>
    --network-interface-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--floating-ip-id` (required)
: Floating IP ID. Run `grn vserver floating-ip list` to browse.

`--network-interface-id` (required)
: Network interface ID to attach the floating IP to. Run `grn vserver server list-interfaces --server-id <id>` to browse.

### Examples

```bash
grn vserver server attach-floating-ip \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --floating-ip-id fip-abc12345-0000-0000-0000-000000000001 \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001
```

---

## detach-floating-ip

Detach a floating IP from a server's network interface.

### Synopsis

```
grn vserver server detach-floating-ip
    --server-id <value>
    --floating-ip-id <value>
    --network-interface-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--floating-ip-id` (required)
: Floating IP ID.

`--network-interface-id` (required)
: Network interface ID.

### Examples

```bash
grn vserver server detach-floating-ip \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --floating-ip-id fip-abc12345-0000-0000-0000-000000000001 \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001
```

---

## list-interfaces

List the network interfaces attached to a vServer instance. In table output, internal and external interfaces are shown in separate tables.

### Synopsis

```
grn vserver server list-interfaces --server-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

### Examples

```bash
grn vserver server list-interfaces \
  --server-id srv-abc12345-0000-0000-0000-000000000001

grn vserver server list-interfaces \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --output table
```

---

## attach-internal-interface

Attach an internal subnet interface to a vServer instance.

### Synopsis

```
grn vserver server attach-internal-interface
    --server-id <value>
    --subnet-id <value>
    [--ip <value>]
```

### Options

`--server-id` (required)
: Server UUID.

`--subnet-id` (required)
: Subnet ID to attach. Run `grn vserver subnet list --vpc-id <vpc-id>` to browse.

`--ip` (string)
: Fixed IP address within the subnet. If omitted, an IP is assigned automatically.

### Examples

```bash
# Auto-assigned IP
grn vserver server attach-internal-interface \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --subnet-id sub-abc12345-0000-0000-0000-000000000001

# Fixed IP
grn vserver server attach-internal-interface \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --subnet-id sub-abc12345-0000-0000-0000-000000000001 \
  --ip 10.0.1.50
```

---

## detach-internal-interface

Detach one or more internal network interfaces from a vServer instance.

### Synopsis

```
grn vserver server detach-internal-interface
    --server-id <value>
    --network-interface-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--network-interface-id` (required)
: Comma-separated list of internal network interface IDs to detach.

### Examples

```bash
grn vserver server detach-internal-interface \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001

# Detach multiple interfaces at once
grn vserver server detach-internal-interface \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --network-interface-id eni-aaa111,eni-bbb222
```

---

## attach-external-interface

Attach an external (elastic) network interface to a vServer instance.

### Synopsis

```
grn vserver server attach-external-interface
    --server-id <value>
    --network-interface-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--network-interface-id` (required)
: External network interface ID. Run `grn vserver network-interface list` to browse.

### Examples

```bash
grn vserver server attach-external-interface \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001
```

---

## detach-external-interface

Detach an external network interface from a vServer instance.

### Synopsis

```
grn vserver server detach-external-interface
    --server-id <value>
    --network-interface-id <value>
```

### Options

`--server-id` (required)
: Server UUID.

`--network-interface-id` (required)
: External network interface ID.

### Examples

```bash
grn vserver server detach-external-interface \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --network-interface-id eni-abc12345-0000-0000-0000-000000000001
```

---

## tag-key

List all tag keys available in the project. Use the returned keys with `tag-value` to discover valid values.

### Synopsis

```
grn vserver server tag-key
```

### Examples

```bash
grn vserver server tag-key
grn vserver server tag-key --output table
```

---

## tag-value

List the possible values for a tag key.

### Synopsis

```
grn vserver server tag-value --key <value>
```

### Options

`--key` (required)
: Tag key to look up values for. Run `grn vserver server tag-key` to see available keys.

### Examples

```bash
grn vserver server tag-value --key env
grn vserver server tag-value --key team --output table
```

---

## delete

Delete a vServer instance. Shows a preview and asks for confirmation unless `--force` is used.

### Synopsis

```
grn vserver server delete
    --server-id <value>
    [--delete-all-volumes]
    [--force]
```

### Options

`--server-id` (required)
: Server UUID.

`--delete-all-volumes` (boolean)
: Also delete all volumes attached to the server.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
# Interactive confirmation
grn vserver server delete --server-id srv-abc12345-0000-0000-0000-000000000001

# Delete server and all its volumes, no prompt
grn vserver server delete \
  --server-id srv-abc12345-0000-0000-0000-000000000001 \
  --delete-all-volumes \
  --force
```

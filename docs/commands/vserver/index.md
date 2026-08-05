# vServer Commands

VNG Virtual Server (vServer) commands for managing cloud virtual machines and related infrastructure.

```bash
grn vserver <resource> <command> [options]
```

## Available resources

| Resource | Description |
|----------|-------------|
| [server](server.md) | Create and manage vServer instances |
| [volume](volume.md) | Manage block storage volumes |
| [vpc](vpc.md) | Manage VPC networks |
| [subnet](subnet.md) | Manage subnets within a VPC |
| [secgroup](secgroup.md) | Manage security groups and their rules |
| [sshkey](sshkey.md) | Manage SSH key pairs |
| [floating-ip](floating-ip.md) | Manage floating (public) IPs |
| [network-interface](network-interface.md) | Manage elastic network interfaces |
| [placement-group](placement-group.md) | Manage server placement groups |
| [dhcp](dhcp.md) | Manage DHCP option sets |
| [user-image](user-image.md) | Manage user-created images |
| [flavor](flavor.md) | Browse available instance flavors |
| [image](image.md) | Browse available OS/GPU images |
| [volume-type](volume-type.md) | Browse available volume types for a zone |

## Quick start workflow

Creating a server requires several prerequisites. Run these commands in order:

```bash
# 1. Create a VPC
grn vserver vpc create --name my-vpc --cidr 10.0.0.0/16

# 2. Create a subnet inside the VPC
grn vserver subnet create --vpc-id <vpc-id> --cidr 10.0.1.0/24 --zone-id <zone-id>

# 3. Find an available zone (omit --zone-id to see the list)
#    Zone IDs are shown when you run any command that requires --zone-id

# 4. Find a flavor
grn vserver flavor list-families
grn vserver flavor list-codes
grn vserver flavor list --family <family> --code <code>

# 5. Find an image
grn vserver image list --type os

# 6. Find a volume type for the zone
grn vserver volume-type list --zone-id <zone-id> --type SSD

# 7. Create the server
grn vserver server create \
  --name my-server \
  --flavor-id <flavor-id> \
  --image-id <image-id> \
  --network-id <vpc-id> \
  --subnet-id <subnet-id> \
  --root-disk-type-id <volume-type-id> \
  --zone-id <zone-id>
```

## Common global options

All vserver commands accept these global flags:

| Flag | Description |
|------|-------------|
| `--region` | Override the configured region |
| `--output` | Output format: `json` (default), `table`, `text` |
| `--query` | JMESPath query to filter output |
| `--profile` | Use a specific credentials profile |
| `--endpoint-url` | Override the vServer API endpoint |
| `--debug` | Print raw HTTP requests and responses |

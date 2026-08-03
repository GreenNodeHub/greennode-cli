# sshkey

Manage SSH key pairs for authenticating to vServer instances.

```bash
grn vserver sshkey <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all SSH keys |
| [create](#create) | Generate a new SSH key pair |
| [import](#import) | Import an existing SSH public key |
| [delete](#delete) | Delete an SSH key |

---

## list

List all SSH keys in your project.

### Synopsis

```
grn vserver sshkey list
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
: Filter by key name (substring match).

### Examples

```bash
grn vserver sshkey list
grn vserver sshkey list --output table
```

---

## create

Generate a new SSH key pair. The private key is saved locally and the public key is registered in your project. Use `--ssh-key-id` when creating a server to inject the key.

### Synopsis

```
grn vserver sshkey create
    --name <value>
    [--output-dir <value>]
```

### Options

`--name` (required)
: SSH key name.

`--output-dir` (string)
: Directory to save the `.pem` private key file. Defaults to the system Downloads folder. If a file named `<name>.pem` already exists, the file is saved as `<name>(1).pem`, `<name>(2).pem`, etc.

### Examples

```bash
# Generate a key and save the .pem to ~/Downloads
grn vserver sshkey create --name my-key

# Save the .pem to a specific directory
grn vserver sshkey create --name deploy-key --output-dir ~/.ssh
```

---

## import

Import an existing SSH public key into your project.

### Synopsis

```
grn vserver sshkey import
    --name <value>
    (--public-key <value> | --public-key-file <value>)
```

### Options

`--name` (required)
: SSH key name.

`--public-key` (string)
: SSH public key string, e.g. `ssh-rsa AAAA...`.

`--public-key-file` (string)
: Path to a local file containing the SSH public key. Exactly one of `--public-key` or `--public-key-file` must be provided.

### Examples

```bash
# Import from a file
grn vserver sshkey import --name my-key --public-key-file ~/.ssh/id_rsa.pub

# Import inline
grn vserver sshkey import --name my-key --public-key "ssh-rsa AAAA..."
```

---

## delete

Delete an SSH key. Shows a confirmation prompt unless `--force` is used.

### Synopsis

```
grn vserver sshkey delete
    --sshkey-id <value>
    [--force]
```

### Options

`--sshkey-id` (required)
: SSH key ID.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
grn vserver sshkey delete --sshkey-id key-abc12345-0000-0000-0000-000000000001
grn vserver sshkey delete --sshkey-id key-abc12345-0000-0000-0000-000000000001 --force
```

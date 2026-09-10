# sshkey

Manage SSH key pairs for authenticating to vServer instances.

```bash
grn vserver sshkey <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all SSH keys |
| [get](#get) | Get an SSH key |
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
    [--show-secret]
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

Private-key fields are `[REDACTED]` unless `--show-secret` is set. Public-key fields remain visible.

---

## get

Get one SSH key by ID.

```bash
grn vserver sshkey get --sshkey-id <id> [--show-secret]
```

Private-key fields are `[REDACTED]` unless `--show-secret` is set. Public-key fields remain visible.

---

## create

Generate a new SSH key pair. The private key is saved locally and the public key is registered in your project. Use `--ssh-key-id` when creating a server to inject the key.

### Synopsis

```
grn vserver sshkey create
    --name <value>
    [--output-dir <value>]
    [--show-secret]
```

### Options

`--name` (required)
: SSH key name.

`--output-dir` (string)
: Directory to save the `.pem` private key file. Defaults to the system Downloads folder. If a file named `<name>.pem` already exists, the file is saved as `<name>(1).pem`, `<name>(2).pem`, etc.

`--show-secret` (boolean)
: Also permit private-key material in command output. The saved file remains mode `0600`; existing paths are not overwritten.

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
    [--show-secret]
```

### Options

`--name` (required)
: SSH key name.

`--public-key` (string)
: SSH public key string, e.g. `ssh-rsa AAAA...`.

`--public-key-file` (string)
: Path to a local file containing the SSH public key. Exactly one of `--public-key` or `--public-key-file` must be provided.

`--show-secret` (boolean)
: Permit any secret fields returned by the API in command output.

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
    [--dry-run]
```

### Options

`--sshkey-id` (required)
: SSH key ID.

`--force` (boolean)
: Skip the confirmation prompt.

`--dry-run` (boolean)
: Preview the deletion without executing it.

### Examples

```bash
grn vserver sshkey delete --sshkey-id key-abc12345-0000-0000-0000-000000000001
grn vserver sshkey delete --sshkey-id key-abc12345-0000-0000-0000-000000000001 --force
grn vserver sshkey delete --sshkey-id key-abc12345-0000-0000-0000-000000000001 --dry-run
```

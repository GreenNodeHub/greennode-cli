# placement-group

Manage placement groups (server groups) to control how vServer instances are distributed across physical hosts.

```bash
grn vserver placement-group <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all placement groups |
| [list-policies](#list-policies) | List available placement policies |
| [create](#create) | Create a new placement group |
| [edit](#edit) | Update a placement group |
| [delete](#delete) | Delete a placement group |

---

## list

List all placement groups in your project.

### Synopsis

```
grn vserver placement-group list
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
: Filter by placement group name (substring match).

### Examples

```bash
grn vserver placement-group list
grn vserver placement-group list --output table
```

---

## list-policies

List available placement group policies. Use this to find a `--policy-id` for `create`.

### Synopsis

```
grn vserver placement-group list-policies
    [--language <value>]
```

### Options

`--language` (string)
: Language for policy descriptions. Accepted values: `en`, `vi`. Default: `en`.

### Examples

```bash
grn vserver placement-group list-policies
grn vserver placement-group list-policies --language vi
```

---

## create

Create a new placement group.

### Synopsis

```
grn vserver placement-group create
    --name <value>
    [--description <value>]
    [--policy-id <value>]
```

### Options

`--name` (required)
: Placement group name.

`--description` (string)
: Placement group description.

`--policy-id` (string)
: Policy ID controlling how servers are placed. If omitted, the command lists available policies and prompts you to choose one. Run `grn vserver placement-group list-policies` to browse options first.

### Examples

```bash
# Interactive policy selection
grn vserver placement-group create --name my-group

# Specify policy directly
grn vserver placement-group create \
  --name anti-affinity-group \
  --description "Spread servers across hosts" \
  --policy-id policy-abc12345
```

---

## edit

Update a placement group's name or description. Only the fields you provide are updated.

### Synopsis

```
grn vserver placement-group edit
    --placement-group-id <value>
    [--name <value>]
    [--description <value>]
```

### Options

`--placement-group-id` (required)
: Placement group ID.

`--name` (string)
: New name for the placement group.

`--description` (string)
: New description.

### Examples

```bash
grn vserver placement-group edit \
  --placement-group-id pg-abc12345-0000-0000-0000-000000000001 \
  --name new-name

grn vserver placement-group edit \
  --placement-group-id pg-abc12345-0000-0000-0000-000000000001 \
  --description "Updated description"
```

---

## delete

Delete a placement group. Shows a confirmation prompt unless `--force` is used.

### Synopsis

```
grn vserver placement-group delete
    --placement-group-id <value>
    [--force]
```

### Options

`--placement-group-id` (required)
: Placement group ID.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
grn vserver placement-group delete \
  --placement-group-id pg-abc12345-0000-0000-0000-000000000001

grn vserver placement-group delete \
  --placement-group-id pg-abc12345-0000-0000-0000-000000000001 \
  --force
```

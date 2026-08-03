# user-image

Manage user-created images (snapshots taken from existing vServer instances via `grn vserver server create-image`).

```bash
grn vserver user-image <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List all user images |
| [update-tags](#update-tags) | Update tags on a user image |
| [delete](#delete) | Delete a user image |

---

## list

List all user images in your project.

### Synopsis

```
grn vserver user-image list
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
: Filter by image name (substring match).

### Examples

```bash
grn vserver user-image list
grn vserver user-image list --output table
grn vserver user-image list --query "listData[].{id: uuid, name: name, status: status}"
```

---

## update-tags

Update the tags on a user image.

### Synopsis

```
grn vserver user-image update-tags
    --user-image-id <value>
    [--tag <value> ...]
    [--edited-tag <value> ...]
```

### Options

`--user-image-id` (required)
: User image ID.

`--tag` (string, repeatable)
: New tag to add in `key=value` format.

`--edited-tag` (string, repeatable)
: Existing tag being modified, in `key=value` format.

### Examples

```bash
grn vserver user-image update-tags \
  --user-image-id img-abc12345-0000-0000-0000-000000000001 \
  --tag env=prod \
  --edited-tag version=v2
```

---

## delete

Delete a user image. Shows a confirmation prompt unless `--force` is used.

### Synopsis

```
grn vserver user-image delete
    --user-image-id <value>
    [--force]
```

### Options

`--user-image-id` (required)
: User image ID.

`--force` (boolean)
: Skip the confirmation prompt.

### Examples

```bash
grn vserver user-image delete --user-image-id img-abc12345-0000-0000-0000-000000000001
grn vserver user-image delete --user-image-id img-abc12345-0000-0000-0000-000000000001 --force
```

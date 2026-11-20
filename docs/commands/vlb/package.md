# package

Browse the packages a load balancer can be created with or resized to.

```bash
grn vlb package <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List load balancer packages |

---

## list

List load balancer packages. By default this shows the packages currently
offered; `--all` includes those no longer on sale, which existing load
balancers may still be on.

### Synopsis

```
grn vlb package list [--zone-id <value>] [--all]
```

### Options

`--zone-id` (string)
: List the active packages for a specific availability zone. Cannot be
  combined with `--all`.

`--all` (boolean)
: List every package, including those not currently offered. Default: `false`.

### Examples

```bash
grn vlb package list --output table
grn vlb package list --zone-id HCM03-1A --output table
grn vlb package list --all --query 'listData[].{id:uuid,name:name,conns:connectionNumber}'
```

---

## Global options

All commands accept the [global options](../../configuration.md): `--profile`,
`--region`, `--output`, `--query`, `--endpoint-url`, `--no-verify-ssl`,
`--debug`, `--color`, `--cli-read-timeout`, `--cli-connect-timeout`,
`--allow-untrusted-endpoint`.

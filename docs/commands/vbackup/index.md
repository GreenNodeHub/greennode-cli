# vBackup Gateway

`grn vbackup` manages the 29 operations in the [official API reference](https://docs.api.greennode.ai/service-docs/vbackup-gateway.html): backup policies, protected servers, restore points, destinations, configuration, history, and volume usage.

## Configuration

Use a machine or user-login profile with region `HCM-3`. The gateway is `https://hcm-3.api.vngcloud.vn/vbackup-gateway`; it uses bearer authentication without a `portal-user-id` header. An explicit `--endpoint-url` bypasses the region gate but still follows endpoint-safety checks.

## Commands

| Group | Commands |
| --- | --- |
| `backend` | `list` |
| `destination` | `list` |
| `policy` | `list`, `get`, `create`, `update`, `delete` |
| `server` | `list`, `get`, `create`, `delete`, `list-protected`, `list-points`, `list-volumes`, `enable`, `disable`, `update-policy`, `update-volume` |
| `vserver` | `list-instances`, `create-instance`, `get-instance`, `list-instance-points`, `get-instance-point`, `list-volume-points`, `get-volume-point` |
| `configuration` | `get` |
| `history` | `list-backups`, `list-restorations` |
| `volume` | `usage` |

Use `grn vbackup <group> <command> --help` for exact flags. List filters are route-specific. Where supported, `--page` starts at 1 and `--size` is a 32-bit integer; unset parameters are omitted and pagination is not automatic.

## Bodies and safety

Body-bearing commands accept `--body '<json>'`. The reference declares objects without required properties; the CLI validates syntax and object shape, while the API validates business fields.

All 10 mutations support offline `--dry-run`. Deletes also require confirmation or `--force`; non-interactive refusal returns an error. Mutations are never retried automatically, including after 401. `volume usage` is a read-only POST and has no `--dry-run`.

Deletes accept empty HTTP 204; enable/disable and server policy/volume updates accept empty HTTP 200. Other successful responses must have the documented status and JSON object/array type.

```bash
grn --region HCM-3 vbackup backend list --page 1 --size 25
grn --region HCM-3 vbackup policy create --body '{"name":"example-policy"}' --dry-run
grn --region HCM-3 vbackup server get --id <backup-instance-id>
grn --region HCM-3 --non-interactive vbackup policy delete --id <policy-id> --force
```

Local contract and integration tests use synthetic data. They do not establish live provider permissions or successful backup lifecycles.

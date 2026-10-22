# vStorage

`grn vstorage` manages storage projects, containers, buckets, and object metadata. Its 97 commands cover 46 HCM03 and 66 HAN02/HCM04 published operations, with 15 shared operations.

| Region | Endpoint | Resource family |
| --- | --- | --- |
| `HCM-3` | `https://hcm03-api.vstorage.vngcloud.vn` | Swift containers |
| `HAN` | `https://han02-api.vstorage.vngcloud.vn` | Ceph/S3 buckets |
| `HCM-4` | `https://hcm04-api.vstorage.vngcloud.vn` | Ceph/S3 buckets |

Use `region`, `project`, and `billing` in any region; `container` only in HCM-3; `bucket` only in HAN/HCM-4. Project deletion and member listing are HCM-3-only. Each project-scoped command requires `--project-id`. An explicit `--endpoint-url` bypasses the region-family check.

## Requests and safety

Use `--help` on each group for its commands. Body-bearing operations require `--body '<json>'` with the published object or array shape; numbers retain their precision. Lifecycle, notification, copy, and move schemas are resolved from the official references. Project creation also accepts `--poc`, adding `isPoc: true`.

Every mutation supports offline `--dry-run`; destructive operations require confirmation or `--force`. Unforced non-interactive deletion fails. Writes are never automatically retried. Search POST requests are read-only and do not expose `--dry-run`.

Object and directory names may contain slashes and spaces and are percent-escaped. Container/bucket names allow 3–235 letters, digits, periods, spaces, underscores, hyphens, or `@`. Lifecycle names allow 5–50 letters, digits, spaces, underscores, or hyphens.

Commands named `generate-*-url` return data-plane URLs; this client does not upload or download object bytes. Treat those URLs as credentials. Pagination follows each operation's flags and is not automatic.

An HTTP 200 failure envelope (`success: false`) or empty HTTP 200 is an error. Documented empty POST/PUT 201 and DELETE 204 responses are accepted.

```bash
grn --region HCM-3 vstorage project list
grn --region HCM-3 vstorage container object get-metadata --project-id <id> --container <name> --object 'reports/summary.txt'
grn --region HAN vstorage bucket create --project-id <id> --bucket <name> --dry-run
grn --region HCM-4 vstorage bucket update-policy --project-id <id> --bucket <name> --body '{}' --dry-run
```

Contract fixtures and local HTTP tests verify request construction, regional boundaries, response handling, and safety; they do not prove live resource lifecycles. Payload references: [HCM03](https://docs.api.greennode.ai/service-docs/vstorage-api.html), [HAN02](https://docs.api.greennode.ai/service-docs/vstorage-han02-api.html), [HCM04](https://docs.api.greennode.ai/service-docs/vstorage-hcm04-api.html).

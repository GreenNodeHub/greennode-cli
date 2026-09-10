# vStorage Gateway

`grn vstorage-gateway` provides three read-only vMonitor-vStorage discovery commands:

| Command | Required flags |
| --- | --- |
| `region list` | None |
| `project list` | `--region-id` |
| `container list` | `--region-id`, `--project-id` |

The client uses the shared machine/user profile and bearer authentication. It requires no configured region, project, or portal-user ID. Query flags retain the API's underscore names on the wire. Responses must be HTTP 200 JSON arrays.

The configured default is `https://vmonitorapis.vngcloud.vn/vstorage-gateway` (plural), while the [published API reference](https://docs.api.greennode.ai/service-docs/vstorage-gateway.html) lists `https://vmonitorapi.vngcloud.vn/vstorage-gateway` (singular). This compatibility difference needs maintainer confirmation; current live support is unverified. Use `--endpoint-url` to select an explicitly approved endpoint.

```bash
grn vstorage-gateway region list
grn vstorage-gateway container list --region-id <id> --project-id <id>
```

Public fixtures and local HTTP tests cover all three contracts, not live provider availability.

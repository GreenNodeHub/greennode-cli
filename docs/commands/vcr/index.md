# vCR Commands

Manage the 23 published repository, image, artifact, and repository-user operations.

vCR uses the global `https://vcr.api.vngcloud.vn` endpoint and the selected profile's IAM token. It does not require a project, region, or `portal_user_id`. The reference requires an Authorization header without specifying its scheme; authenticated provider verification remains a release gate.

| Group | Commands |
|---|---|
| `repository` | `list`, `get`, `create`, `delete`, `attach-users`, `detach-users`, `list-history`, `update-quota`, `list-users` |
| `image` | `list`, `get`, `delete` |
| `artifact` | `list`, `delete` |
| `user` | `list`, `create`, `delete`, `update`, `enable`, `disable`, `list-permissions`, `update-permissions`, `refresh-secret` |

Use `--body '<json>'` for exact request objects. The CLI checks required fields and matching path/body IDs. List commands preserve explicit `--page` and `--size`; they do not auto-paginate.

Every mutation supports offline `--dry-run`. Deletion and secret rotation also require confirmation or `--force`. Secret rotation is a state-changing GET and is never retried. User creation and rotation redact returned credentials unless `--show-secret` is explicit; debug output always suppresses those response bodies.

```bash
grn vcr repository list --page 1 --size 20
grn vcr repository create --body '{"repoName":"example-registry","isPublic":false,"quotaLimit":1}' --dry-run
grn vcr user refresh-secret --user-id <user-id> --dry-run
```

Tests use synthetic local HTTP fixtures for all operations, both profile auth modes, and token rotation. They do not establish provider permissions or successful live lifecycles. Request and response authority: [vCR API reference](https://docs.api.greennode.ai/service-docs/vcr.html).

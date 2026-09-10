# IAM

`grn iam` manages identities and authorization through the global [Accounts API](https://docs.api.greennode.ai/service-docs/accounts-api.html) and [Policies API](https://docs.api.greennode.ai/service-docs/policies-api.html).

| API | Endpoint | Commands |
| --- | --- | ---: |
| Accounts | `https://iamapis.vngcloud.vn/accounts-api` | 69 |
| Policies | `https://iamapis.vngcloud.vn/policies-api` | 34 |

Existing `grn configure`, `grn login`, and `grn logout` remain the authentication interfaces. IAM management uses both shared machine/user profile providers and refresh-token rotation. No configured region, project, or portal-user ID is required.

## Command groups

| Group | Operations |
| --- | --- |
| `whoami` | Current identity |
| `identity-provider` | Providers and permission mappers |
| `service-account` | Lifecycle, secret reset, S3/Swift attachments, tags, trusted roots, policies |
| `iam-user` | Lifecycle, passwords, email and Authenticator MFA, tags, groups, policies |
| `s3-key`, `swift-user` | Storage credentials |
| `group`, `policy` | CRUD, tags, bindings, policy composition |
| `action`, `product`, `resource` | Authorization vocabulary |

Use each group's `--help` for the full command list. No new token-issuance, impersonation, OAuth2, browser, callback, or OAuth-link command is added. Google commands manage Authenticator MFA, not OAuth sign-in.

## Safety

Every mutation requires confirmation or `--force` and supports offline `--dry-run`. Dry-runs validate inputs and read supplied body files but do not load credentials, create clients, prompt, call APIs, or write files. Direct IAM-user/service-account mutations check `userinfo` and refuse the current identity; dedicated `iam-user current` commands are separate. Unknown identity types fail closed.

Mutations reject `--endpoint-url`; reads retain shared endpoint checks. IAM requests are not retried automatically. Binding responses require empty HTTP 204; other mutations enforce their operation-specific status and body contract.

Body-bearing commands accept `--body` or `--body-file`, validate the published top-level shape and required fields, and preserve other fields. Credential-bearing bodies require a regular, non-symlink `--body-file` without group/other permissions. Policy composition takes a JSON array.

Credential reads and secret-bearing mutation responses default to `[REDACTED]`; `--show-secret` explicitly permits command output only. Debug and API-error response bodies remain suppressed even with this flag. Treat captured output as sensitive.

## Compatibility notes

Paginated lists use zero-based `--page-number` (default 0) and positive `--page-size` (default 10), without automatic traversal. Service-account listing also sends these values as a compatibility exception: its published route omits pagination parameters.

Authenticator setup publishes an empty HTTP 200 but may return an undocumented provisioning payload. Such payloads are fully masked unless `--show-secret` is supplied. Service-account creation discards undocumented success payloads. These exceptions need provider confirmation; no live lifecycle or MFA enrollment was performed.

```bash
grn iam whoami
grn iam iam-user list --page-number 0 --page-size 10
grn iam group create --body '{"name":"example-group"}' --dry-run
grn iam iam-user create --body-file private-user.json --dry-run
```

Independent fixtures pin 72 Accounts and 34 Policies contracts. The command surface excludes Accounts token issuance and impersonation; IAM v2 token acquisition remains internal. Local HTTP tests cover accepted routes, both auth modes, retry refusal, protected body files, and secret-output boundaries.

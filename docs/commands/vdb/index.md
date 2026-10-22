# vDB Commands

Manage 139 published operations across four families.

| Family | Operations | Groups |
|---|---:|---|
| `relational` | 45 | `instance`, `backup`, `backup-storage`, `configuration`, `catalog` |
| `memory` | 44 | `instance`, `backup`, `backup-storage`, `configuration`, `catalog` |
| `kafka` | 36 | `cluster`, `topic`, `user`, `configuration`, `catalog` |
| `postgresql` | 14 | `cluster`, `backup` |

vDB uses `https://vdb-gateway.vngcloud.vn` and the selected profile's IAM token; the CLI corrects the reference's malformed `https:/` scheme. No project prefix or regional endpoint is added.

A positive 32-bit `portal_user_id` is required on 136 operations. MemoryStore configuration detail, MemoryStore instance list, and relational zone list omit it. Only documented operations expose `--user-type ROOT_USER|IAM_USER`.

```bash
grn configure set portal_user_id <numeric-id>
grn vdb kafka cluster list-clusters
grn vdb relational instance get-database-instances-by-user --filter '{}' --page-number 1 --page-size 20
grn vdb kafka topic create-topic --cluster-id <cluster-id> --body '{"name":"example-topic"}' --dry-run
```

JSON object queries are compactly encoded under their exact wire names. Pagination is operation-specific and never automatic. Bodies use `--body '<json>'`; the CLI checks shape, published required fields, action values, and path/body bindings.

Eleven operations expose `--poc` where published schemas permit it. This sets the documented PoC field and `user-type: IAM_USER`; `ROOT_USER` conflicts. Nested resize/restore bodies require `databaseInstances[].config`. A preview cannot confirm payment eligibility.

All 66 mutations support offline `--dry-run` and are never retried. Deletion, restore, detach, reboot, stop, and credential regeneration require confirmation or `--force`. Kafka credential responses are debug-suppressed and redacted unless `--show-secret` is explicit.

Tests use synthetic local HTTP fixtures for all operations, both profile auth modes, and operation-specific headers. They do not prove live provisioning, billing, or permissions. OpenSearch is outside the published contract. See the [vDB API reference](https://docs.api.greennode.ai/service-docs/vdb-api.html) and group `--help` for exact fields.

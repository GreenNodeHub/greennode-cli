# vMonitor Log

`grn vmonitor-log` exposes 63 operations from the [official API reference](https://docs.api.greennode.ai/service-docs/log-api.html).

| Groups | Purpose |
|---|---|
| `project`, `log` | Projects, search, and exports |
| `pipeline`, `processor`, `processor-group`, `processor-group-library` | Processing pipelines |
| `archive`, `refill` | Archive and refill workflows |
| `certificate` | Certificate downloads |
| `vcdn-mapping`, `vdb-mapping`, `vlb-mapping`, `vstorage-mapping`, `vstorage-bucket-mapping` | Product log mappings |

Commands use the shared machine or user profile and the global `https://vmonitorapis.vngcloud.vn/log-api` endpoint, without a regional selection or `portal-user-id` header. The published specification still names the singular `vmonitorapi` host; the CLI retains the plural-host compatibility endpoint. This difference is preserved in the public fixtures and is not a claim of live verification for this contribution.

Path and query flags use kebab-case; requests preserve names such as `sortBy`, `billing_status`, and `region-id`. API `query` is exposed as `--search`, leaving global `--query` for output filtering. CDN domains and bucket names use resource-specific validation. Paging is explicit, with no automatic traversal.

Supply request objects with `--body`; consult the operation reference for required fields. The CLI validates JSON structure rather than the full provider schema. The 35 mutations support offline `--dry-run`; six deletes require confirmation or `--force`. Writes are never automatically retried. Log searches, connection tests, and GROK debugging are read-only POSTs. Search results preserve the provider's stringified JSON.

Archive and refill responses redact credential fields unless `--show-secret` is explicit; debug output and errors stay redacted.

```bash
grn vmonitor-log project list --search errors --page 0 --size 20
grn vmonitor-log pipeline create --body '{"name":"example-pipeline"}' --dry-run
grn vmonitor-log certificate download --project-id fixture-project --cert-id fixture-certificate --output-file certificate.zip --dry-run
```

Certificate downloads require `--output-file`, save bytes atomically with owner-only permissions, and refuse empty responses or symlink/non-regular destinations. Existing files require confirmation or `--force`; new destinations use no-clobber creation. Offline previews neither create clients nor write files.

Contract and local-server tests cover all operations through both authentication modes, including secret handling and binary output. They do not prove live provider permissions or resource lifecycles.

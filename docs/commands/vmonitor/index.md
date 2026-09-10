# vMonitor

`grn vmonitor` exposes 101 operations from the [official API reference](https://docs.api.greennode.ai/service-docs/vmonitor-api.html).

| Groups | Purpose |
|---|---|
| `alarm`, `change-alarm` | Alarm rules and history |
| `api-key` | Metric API keys |
| `dashboard`, `view`, `variable`, `widget`, `widget-v2` | Dashboard composition |
| `infrastructure`, `integration` | Monitored hosts and integrations |
| `metric`, `metric-unit`, `metric-unit-mapping`, `metric-unit-mapping-user` | Metric metadata |
| `statistic` | Statistics queries |

Commands use the shared machine or user profile and the global `https://vmonitorapis.vngcloud.vn/vmonitor-api` endpoint. No regional selection or `portal-user-id` header is added. The published server URL has a malformed `//https://` prefix; the CLI uses its valid HTTPS form.

Path and query flags use kebab-case; requests preserve the published names. All path placeholders are required, including dashboard IDs omitted from some published widget parameter lists. Metric-key and infrastructure host lists default to `page=0&size=50`; explicit flags override these CLI compatibility defaults. Other lists retain their operation-specific behavior.

Supply request objects with `--body`. The CLI validates JSON structure, not every provider field. Mutations support offline `--dry-run`; deletes and `integration uninstall` require confirmation or `--force`. Non-interactive refusal returns an error. `statistic get-v2` is a read-only POST.

Metric-key responses are redacted unless `--show-secret` is explicit. Debug output and errors remain redacted. Key-bearing delete paths are also masked in previews, confirmation prompts, and debug logs. Writes are never automatically retried.

```bash
grn vmonitor dashboard list
grn vmonitor dashboard create --body '{"name":"example-dashboard"}' --dry-run
grn vmonitor api-key list-metric
```

Contract and local-server tests cover all operations through both authentication modes; they do not prove live provider permissions or resource lifecycles.

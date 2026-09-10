# reset-workspace-service-account

Reset the current workspace service account. This can invalidate existing consumers, so confirmation or `--force` is required.

Source: [VKS API](https://docs.api.greennode.ai/service-docs/vks-api.html), `POST /v1/workspace/reset-service-account`.

```bash
grn vks reset-workspace-service-account [--dry-run] [--force]
```

# stop-poc

End proof-of-concept mode for a cluster. This command does not create or manage PoC credits. Confirmation or `--force` is required.

Source: [VKS API](https://docs.api.greennode.ai/service-docs/vks-api.html), `POST /v1/clusters/{clusterId}/stop-poc`.

```bash
grn vks stop-poc --cluster-id <cluster-id> [--dry-run] [--force]
```

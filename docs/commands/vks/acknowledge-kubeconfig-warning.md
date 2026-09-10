# acknowledge-kubeconfig-warning

Acknowledge a cluster's kubeconfig renewal warning. The request has no body and returns HTTP 204.

Source: [VKS API](https://docs.api.greennode.ai/service-docs/vks-api.html), `PUT /v1/clusters/{clusterId}/kubeconfig/acknowledge-warning`.

```bash
grn vks acknowledge-kubeconfig-warning --cluster-id <cluster-id> [--dry-run]
```

# get-nodegroup-events

Get a node group's events. Pagination is 0-based and returns one page.

Source: [VKS API](https://docs.api.greennode.ai/service-docs/vks-api.html), `GET /v1/clusters/{clusterId}/node-groups/{nodeGroupId}/events`.

```bash
grn vks get-nodegroup-events --cluster-id <cluster-id> --nodegroup-id <nodegroup-id> [--action <action>] [--type <type>] [--page <page>] [--page-size <size>]
```

# register-fleet

Register a cluster with fleet management. Although the published schema labels `fleetType` as an object, its enum and examples are the strings `NEW` and `EXISTING`; optional traffic flags accept `enabled` or `disabled`.

Source: [VKS API](https://docs.api.greennode.ai/service-docs/vks-api.html), `POST /v1/clusters/{clusterId}/register-fleet`.

```bash
grn vks register-fleet --cluster-id <cluster-id> --fleet-type NEW|EXISTING [--fleet-id <fleet-id>] [--fleet-name <fleet-name>] [--enable-east-west-traffic enabled|disabled] [--enable-north-south-traffic enabled|disabled] [--dry-run]
```

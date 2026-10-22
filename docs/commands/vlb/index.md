# vLB Commands

Manage the 37 published certificate, load-balancer, listener, L7-policy, and pool operations.

Use `--region HCM-3` or `--region HAN`; HCM-4 has no published gateway. Every operation requires `--project-id`. Authentication uses the selected profile's IAM token, without `portal-user-id`.

`load-balancer list` requires JSON-object `--filter` and `--page-request` flags, sent as `filter` and `pageRequest`. Other query names and pagination remain operation-specific.

```bash
grn --region HCM-3 vlb load-balancer list --project-id <project-id> --filter '{}' --page-request '{"page":0,"size":20}'
grn --region HAN vlb listener list --project-id <project-id> --load-balancer-id <load-balancer-id>
grn --region HCM-3 vlb pool delete --project-id <project-id> --load-balancer-id <load-balancer-id> --pool-id <pool-id> --dry-run
```

Body-bearing writes accept `--body '<json>'`. The CLI checks object shape and published top-level required fields; nested constraints remain provider-validated. Certificate previews redact private keys and passphrases.

Every write supports offline `--dry-run` and is sent once. Deletions require confirmation or `--force`; non-interactive refusal returns an error. Responses must use the published status and body requirement; application envelopes with `success: false` fail.

Tests use synthetic local HTTP fixtures for all operations and both profile auth modes. Live resource lifecycles and undocumented status variations are not verified. See the [vLB API reference](https://docs.api.greennode.ai/service-docs/vlb-api.html) and group `--help` for exact contracts.

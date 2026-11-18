# VKS CLI — product notes

Tier-2 notes for the `grn vks` command group. The root `CLAUDE.md` has the
repo-wide conventions; this file records what is specific to VKS.

## Client & endpoint

- `createClient(cmd)` → `cli.NewClient(cmd, "vks")`. The endpoint is resolved from
  `internal/config` REGIONS key `vks_endpoint`
  (`https://vks.api.vngcloud.vn` HCM-3, `https://vks-han-1.api.vngcloud.vn` HAN).
- Auth: shared GreenNode IAM bearer token (see `internal/auth`).

## API quirks

- **Pagination is 0-based**: params `page` (from 0) + `pageSize`. The `list-*`
  commands default `--page -1`, a sentinel meaning "auto-paginate all pages".
- Paths are under `/v1/clusters/...`.
- Cluster/node-group **status** values: `CREATING`, `ACTIVE`, `ERROR` (waiters key
  on these; see `wait.go`).
- **Kubeconfig is asynchronous**: `generate-kubeconfig` (POST) requests generation,
  then `update-kubeconfig` (GET) fetches + merges once status is `ACTIVE`
  (`NONE`/`CREATING`/`ERROR` are the other states).

## Command specifics

- `create-cluster` creates the **control plane only**. The API's `nodeGroups` array
  is deprecated — do not send it; add workers with `create-nodegroup`.
- `update-nodegroup`: all body fields optional. Labels/tags/taints are **deprecated**
  on this endpoint — use `update-nodegroup-metadata`.
- `config-auto-upgrade` (the former `set-auto-upgrade-config` is a deprecated alias).
- `--k8s-version` values come from `list-cluster-versions`.
- Network types: `TIGERA`, `CILIUM_OVERLAY`, `CILIUM_NATIVE_ROUTING`; `--cidr` is
  required for `TIGERA`/`CILIUM_OVERLAY`. OS images: `ubuntu`, `linux`, `rocky`.

## Struct-valued flags (shorthand `k=v,k2=v2` or JSON)

- `--auto-scale` → `autoScaleConfig` (int: `minSize`, `maxSize`)
- `--upgrade-config` → `upgradeConfig` (int: `maxSurge`, `maxUnavailable`; string `strategy`)
- `--placement-group` → `placementGroupConfigDto` (strings)
- `--auto-upgrade-config` → `autoUpgradeConfig` (`weekdays` — use JSON for multiple days; `time`)
- `--auto-healing-config` → `autoHealingConfig` (bool `enableAutoHealing`, int `timeoutUnhealthy`)

## Add-ons (EKS-style API)

- Paths: `/v1/clusters/{id}/addons[/{name}[/update]]`, catalog `/v1/addon-versions`,
  schema `/v1/addons/{name}/versions/{version}/configuration-schema`, updates
  `/v1/clusters/{id}/updates[/{updateId}]`. Responses are wrapped: `{addon:…}`,
  `{update:…}`, `{addons:[names], addonDetails:[…]}` (details only with `?detail=true`).
- Add-on status: `CREATING`, `ACTIVE`, `DEGRADED`, `UPDATING`, `DELETING`,
  `CREATE_FAILED`, `UPDATE_FAILED`, `DELETE_FAILED`. **`DEGRADED` ≠ failed**: the Helm
  release is applied but workloads are not ready (e.g. no nodes yet); a 2-minute sync
  turns it `ACTIVE`. `wait addon-active` keeps polling on it (EKS fails on it).
- `update-addon` is asynchronous and returns an **update** (`InProgress` → `Successful` |
  `Failed` | `Cancelled`). Omitted fields keep the current value; `configurationValues: ""`
  clears it — so the body holds only flags the user set (`cmd.Flags().Changed`).
- `get-update` / `list-updates` scope must match: `addonName` for add-on updates,
  `nodeGroupId` for node-group updates, none for the cluster. A wrong scope is a 404.
- `list-updates` pages with `maxResults` (1-100) + `nextToken`, not `page`/`pageSize`.
- Add-on versions look like `1.2.0-vks.1` (regex in `addon_helpers.go`, mirrors vks-api
  `AddonConst`); `--k8s-version` for the catalog is sent without a leading `v`.
- `create-addon` is accepted while the cluster is `WAITING_*`/`CREATING`/`ACTIVE`.
- `configurationValues` is a JSON or YAML **string** (≤64 KiB); `file://<path>` is read
  by the CLI.

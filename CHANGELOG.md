# Changelog

## [1.11.0](https://github.com/GreenNodeHub/greennode-cli/compare/v1.10.0...v1.11.0) (2026-08-04)


### Features

* **login:** add login method using PKCE + change iam v2 endpoint ([#12](https://github.com/GreenNodeHub/greennode-cli/issues/12)) ([eb1fef9](https://github.com/GreenNodeHub/greennode-cli/commit/eb1fef96c809c0a5bd52ada9786e67d707f5a74e))

## [1.10.0](https://github.com/GreenNodeHub/greennode-cli/compare/v1.9.1...v1.10.0) (2026-08-03)


### Features

* **vserver:** vserver cli ([#9](https://github.com/GreenNodeHub/greennode-cli/issues/9)) ([9260b23](https://github.com/GreenNodeHub/greennode-cli/commit/9260b23483902c7147b84d8347f3238deb96f3a2))

## [1.9.1](https://github.com/GreenNodeHub/greennode-cli/compare/v1.9.0...v1.9.1) (2026-07-27)


### Documentation

* point mkdocs site/repo URLs to GreenNodeHub ([#3](https://github.com/GreenNodeHub/greennode-cli/issues/3)) ([1a20649](https://github.com/GreenNodeHub/greennode-cli/commit/1a2064959a23c19e863273941b3d79f0c56b6a20))

## [1.9.0](https://github.com/GreenNodeHub/greennode-cli/compare/v1.8.0...v1.9.0) (2026-07-27)


### Features

* **vks:** send grn-vks-cli User-Agent on every VKS API request ([#1](https://github.com/GreenNodeHub/greennode-cli/issues/1)) ([a0ffab6](https://github.com/GreenNodeHub/greennode-cli/commit/a0ffab6398cb043e42314adbe26976abaad82985))

## [1.8.0](https://github.com/vngcloud/greennode-cli/compare/v1.7.3...v1.8.0) (2026-07-25)


### Features

* **vks:** send grn-vks-cli User-Agent on every VKS API request ([#47](https://github.com/vngcloud/greennode-cli/issues/47)) ([cf1e112](https://github.com/vngcloud/greennode-cli/commit/cf1e112ec8c3f465b55e420fb19ca473ed867099))

## [1.7.3](https://github.com/vngcloud/greennode-cli/compare/v1.7.2...v1.7.3) (2026-07-25)


### Bug Fixes

* correct GreenNode spelling (greenode -&gt; greennode) ([#45](https://github.com/vngcloud/greennode-cli/issues/45)) ([9c9f847](https://github.com/vngcloud/greennode-cli/commit/9c9f847c8953cf2ce6af8f5626a26d94c710c0ec))
* **vks:** --secondary-subnets takes CIDRs, not subnet IDs ([#44](https://github.com/vngcloud/greennode-cli/issues/44)) ([233cc4c](https://github.com/vngcloud/greennode-cli/commit/233cc4cda84fd83510224ac203c0c9077f030e7c))

## [1.7.2](https://github.com/vngcloud/greennode-cli/compare/v1.7.1...v1.7.2) (2026-07-07)


### Bug Fixes

* **vks:** validate network-type and auto-healing constraints found in real API calls ([#42](https://github.com/vngcloud/greennode-cli/issues/42)) ([810afee](https://github.com/vngcloud/greennode-cli/commit/810afee445bdfc5366b30d2553b1c67db1e9301b))

## [1.7.1](https://github.com/vngcloud/greennode-cli/compare/v1.7.0...v1.7.1) (2026-07-07)


### Bug Fixes

* **vks:** make update-cluster fields optional per spec ([#40](https://github.com/vngcloud/greennode-cli/issues/40)) ([9423eed](https://github.com/vngcloud/greennode-cli/commit/9423eed78aab1d3342cbd26ce672b744726cfacf))

## [1.7.0](https://github.com/vngcloud/greennode-cli/compare/v1.6.0...v1.7.0) (2026-07-07)


### Features

* scaffold command for new product CLIs ([#32](https://github.com/vngcloud/greennode-cli/issues/32)) ([9e1b4cb](https://github.com/vngcloud/greennode-cli/commit/9e1b4cbd222f9a7e7861596b30f1d177555032b4))


### Documentation

* per-product CLAUDE.md (tier 2) and agent skills ([#35](https://github.com/vngcloud/greennode-cli/issues/35)) ([3a1b7cd](https://github.com/vngcloud/greennode-cli/commit/3a1b7cd4df227dbb42aadc4dca2a905174011835))
* rebrand VNG Cloud to GreenNode in VKS and shared text ([#38](https://github.com/vngcloud/greennode-cli/issues/38)) ([d0fd019](https://github.com/vngcloud/greennode-cli/commit/d0fd0195c2d4643fac60231f7db5ed18b65b9b24))
* **vks:** AWS-CLI-style reference format for all commands ([#30](https://github.com/vngcloud/greennode-cli/issues/30)) ([1fe168d](https://github.com/vngcloud/greennode-cli/commit/1fe168deb60cb4bdefd162abc55594db2a8d879c))

## 1.6.0

### Features
* **vks**: create-cluster gains the remaining optional CreateClusterDto fields: --secondary-subnets, --list-subnet-ids, --node-netmask-size, --service-endpoint, --az-strategy, and struct-valued --auto-upgrade-config/--auto-healing-config (shorthand or JSON, matching create-nodegroup). --subnet-id is now optional (per spec; pass --subnet-id or --list-subnet-ids or neither, the server validates)

### Enhancements
* **core**: cli.ParseStructFlag gains bool-field coercion (ParseStructFlagTyped) so struct flags can carry boolean values in shorthand

## 1.5.0

### Enhancements
* **core**: Guard --endpoint-url against sending the IAM bearer token to untrusted hosts (SEC-08): hosts outside vngcloud.vn/greenode.ai are warned over TLS, and blocked when there is no TLS protection (plain http or --no-verify-ssl) unless --allow-untrusted-endpoint is set

### Bug Fixes
* **core**: Redact credential fields (embedded kubeconfig, tokens, client certs/keys, secrets) in --debug request/response logging so 'update-kubeconfig --debug' and similar no longer print long-lived credentials to stderr (SEC-07)

### API Changes
* **vks**: create-cluster now provisions the control plane only and no longer creates a default node group: removed --node-group-name/--flavor-id/--os/--disk-type/--ssh-key-id/--disk-size/--num-nodes/--private-nodes/--security-groups/--labels/--taints. Create workers separately with create-nodegroup

## 1.4.1

### Enhancements
* **vks**: Add --dry-run to the remaining mutating VKS commands (config-auto-healing, config-auto-upgrade, update-nodegroup-metadata, upgrade-nodegroup-version, generate-kubeconfig); it previews the request payload and exits without calling the API (works offline)

## 1.4.0

### Features
* **output**: Implement the --color flag (on/off/auto, like aws): color status values (ACTIVE=green, ERROR/FAILED=red, CREATING/PENDING=yellow) in text and table output. auto colors only when stdout is a terminal and NO_COLOR is unset; JSON output is never colored

### Enhancements
* **core**: Add --dry-run to destructive commands that lacked it (vServer server stop/reboot/delete, volume/vpc/subnet/secgroup/secgroup-rule delete, and vks delete-auto-upgrade-config) and unify the preview + confirmation flow across delete commands via shared helpers

### Bug Fixes
* **configure**: Mask credential values in 'configure set' output so client_id/client_secret are no longer echoed in plaintext to stdout (consistent with 'configure list'); non-sensitive values are still shown
* **vks**: grn vks wait now aborts immediately on a permanent error (HTTP 403/401/400, or 404 for an active waiter) instead of polling until timeout; transient errors (5xx, network) still retry

### API Changes
* **vks**: Rename 'set-auto-upgrade-config' to 'config-auto-upgrade'; the old name still works as a deprecated alias

## 1.3.1

### Enhancements
* **core**: On error, print a single concise 'Error: ...' line instead of cobra's duplicated error plus a full usage dump (use 'grn <command> --help' for usage)
* **vks**: Document the --unhealthy-range format in config-auto-healing help: expects "[min-max]" (e.g. "[2-5]")

### Bug Fixes
* **configure**: Fix panic (nil pointer) when running 'grn configure set/list --profile <name>' for a profile that does not exist yet. LoadConfig now reads the credentials and config files independently, so a profile created via 'configure set' (config file only) loads its region/output/project_id correctly, and 'configure get'/'configure list' on a truly missing profile report a clear 'profile does not exist' error (exit 1, like 'aws configure') instead of crashing
* **core**: Honor --cli-connect-timeout: the TCP connect and TLS handshake are now bounded by the flag (previously it was accepted but ignored, so a slow/unreachable endpoint hung for ~127s regardless). Wired via the HTTP transport's dialer in both VKS and vServer clients
* **output**: Reject an invalid --output or --color value (e.g. a typo like 'tabel') with a clear error, a 'maybe you meant' suggestion, and a non-zero exit, instead of silently falling back

## 1.3.0

### Features
* **vks**: Add struct-valued flags (shorthand or JSON) to create-nodegroup: --tags, --secondary-subnets, --auto-scale, --placement-group, --upgrade-config

### API Changes
* **vks**: update-nodegroup: drop deprecated --labels/--taints (use update-nodegroup-metadata); replace --auto-scale-min/max and --upgrade-strategy/max-surge/max-unavailable with struct flags --auto-scale and --upgrade-config

## 1.2.0

### Bug Fixes
* **configure**: Fix panic (nil pointer) when running 'grn configure --profile <name>' for a profile that does not exist yet; start from empty defaults so the profile can be created
* **vks**: Return a non-zero exit code and 'unknown command' error for invalid subcommands of grn vks/vserver (and their groups) instead of silently printing help with exit 0

### API Changes
* **vks**: Remove get-nodegroup-events; network-type CALICO->TIGERA; --os adds rocky; replace boolean toggle flags with --private-cluster/--private-nodes/--load-balancer-plugin/--block-store-csi-plugin <enabled|disabled>

## 1.1.0

### Features
* **vserver**: Add vServer CLI: vpc, subnet, secgroup, volume, volume-type, flavor, image, and server commands

### API Changes
* **vks**: Align node group commands with latest API: remove --image-id (no longer in the API), add --os (ubuntu/linux, default ubuntu) for create-cluster/create-nodegroup; update-nodegroup no longer requires/sends image

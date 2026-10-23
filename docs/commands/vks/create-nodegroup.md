# create-nodegroup

## Description

Create a new node group within an existing VKS cluster. By default, one node is created with an Ubuntu OS image, a 100 GiB disk, and a SURGE upgrade strategy.

Node group names must be 5–15 characters, contain only lowercase letters, digits, and hyphens, and must start and end with a letter or digit.

Use `--dry-run` to validate parameters (name format, disk size range, node count range) without sending a create request.

## Synopsis

```
grn vks create-nodegroup
    --cluster-id <value>
    --name <value>
    --flavor-id <value>
    --disk-type <value>
    --ssh-key-id <value>
    [--os <value>]
    [--private-nodes <enabled|disabled>]
    [--num-nodes <value>]
    [--disk-size <value>]
    [--security-groups <value>]
    [--subnet-id <value>]
    [--labels <value>]
    [--taints <value>]
    [--tags <value>]
    [--secondary-subnets <value>]
    [--auto-scale <value>]
    [--placement-group <value>]
    [--upgrade-config <value>]
    [--enable-encryption-volume]
    [--dry-run]
```

## Options

**`--cluster-id`** (string)

ID of the cluster to add the node group to.

- Required: Yes

**`--name`** (string)

Name of the node group.

- Required: Yes
- Constraints: 5–15 characters; lowercase letters, digits, and hyphens; must start and end with a letter or digit.

**`--flavor-id`** (string)

Flavor (instance type) ID for the nodes.

- Required: Yes

**`--disk-type`** (string)

Disk type ID for the node boot volumes (e.g. `SSD`, `NVME`).

- Required: Yes

**`--ssh-key-id`** (string)

SSH key pair ID to inject into each node.

- Required: Yes

**`--os`** (string)

Operating system image for the nodes.

- Required: No
- Default: `ubuntu`
- Possible values: `ubuntu`, `linux`, `rocky`

**`--private-nodes`** (string)

Private nodes state. `enabled` means nodes will not have public IP addresses.

- Required: No
- Default: `disabled`
- Possible values: `enabled`, `disabled`

**`--num-nodes`** (integer)

Number of nodes to create in the node group.

- Required: No
- Default: `1`
- Constraints: 0–10

**`--disk-size`** (integer)

Boot disk size in GiB.

- Required: No
- Default: `100`
- Constraints: 20–5000

**`--security-groups`** (list&lt;string&gt;)

Security group IDs to attach to the nodes, comma-separated.

- Required: No
- Syntax: `secg-aaa111,secg-bbb222`

**`--subnet-id`** (string)

Subnet ID for the node group. Uses the cluster subnet when not specified.

- Required: No

**`--labels`** (map)

Kubernetes node labels as comma-separated `key=value` pairs.

- Required: No
- Syntax: `env=prod,tier=app`

**`--taints`** (list&lt;string&gt;)

Kubernetes node taints as comma-separated `key=value:effect` entries.

- Required: No
- Syntax: `dedicated=gpu:NoSchedule`

**`--tags`** (map)

Cloud tags for the node group as comma-separated `key=value` pairs.

- Required: No
- Syntax: `team=platform,cost-center=42`

**`--secondary-subnets`** (list&lt;string&gt;)

Secondary subnet IDs for the node group, comma-separated.

- Required: No
- Syntax: `sub-aaa,sub-bbb`

**`--auto-scale`** (structure)

Auto-scaling configuration for the node group. Accepts shorthand or JSON.

- Required: No
- Members:
    - `minSize` (integer) — minimum number of nodes; minimum value `0`
    - `maxSize` (integer) — maximum number of nodes; minimum value `1`

Shorthand syntax:

```
minSize=2,maxSize=10
```

JSON syntax:

```json
{"minSize": 2, "maxSize": 10}
```

- When given, **both** `minSize` and `maxSize` are required and must be integers,
  with `minSize` ≥ 0, `maxSize` ≥ 1, and `minSize` not exceeding `maxSize`.
  Checked before the request is sent.

**`--placement-group`** (structure)

Placement group configuration for the node group. Accepts shorthand or JSON.

- Required: No
- Members:
    - `type` (string) — `NEW` to create a new placement group, `EXISTING` to use an existing one
    - `placementGroupId` (string) — ID of an existing placement group; used when `type` is `EXISTING`
    - `placementGroupName` (string) — name for a new placement group; used when `type` is `NEW`

Shorthand syntax:

```
type=NEW,placementGroupName=pg-1
```

JSON syntax:

```json
{"type": "EXISTING", "placementGroupId": "server-group-06b86747-eaf7-47dd-9e41-579c2e30bfdd"}
```

**`--upgrade-config`** (structure)

Upgrade strategy configuration for the node group. Accepts shorthand or JSON.

- Required: No
- Default: `maxSurge=1,maxUnavailable=0,strategy=SURGE`
- Members:
    - `strategy` (string) — upgrade strategy; currently only `SURGE` is supported
    - `maxSurge` (integer) — maximum number of extra nodes added during upgrade; range 1–100
    - `maxUnavailable` (integer) — maximum number of nodes that may be unavailable during upgrade; range 0–100

Shorthand syntax:

```
maxSurge=1,maxUnavailable=0,strategy=SURGE
```

JSON syntax:

```json
{"maxSurge": 1, "maxUnavailable": 0, "strategy": "SURGE"}
```

- A partial value is completed rather than replacing the whole object: omitted or
  `null` members fall back to `strategy=SURGE`, `maxSurge=1`, `maxUnavailable=0`.
  `strategy` is required by the API, so this is what keeps `--upgrade-config
  maxSurge=2` a valid request.
- Bounds are checked client-side: `maxSurge` 1–100, `maxUnavailable` 0–100.

**`--enable-encryption-volume`** (boolean)

Enable encryption for the node boot volumes.

- Required: No
- Default: `false`

**`--dry-run`** (boolean)

Validate parameters and print a report without sending the create request.

- Required: No
- Default: `false`

## Global options

This command also accepts the global options (`--profile`, `--region`, `--output`, `--query`, `--endpoint-url`, `--debug`, …).

## Examples

Create a basic node group:

```bash
grn vks create-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --name worker-ng \
  --flavor-id flv-4c8g \
  --disk-type SSD \
  --ssh-key-id key-abc12345-0000-0000-0000-000000000001
```

Create a GPU node group with taints and labels:

```bash
grn vks create-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --name gpu-ng \
  --flavor-id flv-8c32g-gpu \
  --disk-type SSD \
  --disk-size 200 \
  --num-nodes 2 \
  --ssh-key-id key-abc12345-0000-0000-0000-000000000001 \
  --labels accelerator=nvidia,tier=gpu \
  --taints dedicated=gpu:NoSchedule \
  --enable-encryption-volume
```

Create an auto-scaling node group with a custom upgrade config:

```bash
grn vks create-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --name auto-ng \
  --flavor-id flv-4c8g \
  --disk-type SSD \
  --ssh-key-id key-abc12345-0000-0000-0000-000000000001 \
  --auto-scale minSize=2,maxSize=10 \
  --upgrade-config '{"maxSurge":2,"maxUnavailable":1,"strategy":"SURGE"}' \
  --tags team=platform,env=prod
```

Validate parameters without creating (dry run):

```bash
grn vks create-nodegroup \
  --cluster-id cls-abc12345-6789-def0-1234-abcdef012345 \
  --name worker-ng \
  --flavor-id flv-4c8g \
  --disk-type SSD \
  --ssh-key-id key-abc12345-0000-0000-0000-000000000001 \
  --dry-run
```

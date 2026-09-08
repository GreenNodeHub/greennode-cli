# VLB Commands

VNG Load Balancer (VLB) commands for managing load balancers, their listeners,
backend pools, members and certificates.

```bash
grn vlb <resource> <command> [options]
```

All VLB commands act on the project configured by `grn configure` (or
`GRN_DEFAULT_PROJECT_ID`) in the selected region.

## Available resources

| Resource | Description |
|----------|-------------|
| [loadbalancer](load-balancer.md) | Create, resize, clone and delete load balancers |
| [listener](listener.md) | Manage listeners and their L7 routing policies |
| [pool](pool.md) | Manage backend pools, their health monitors and members |
| [certificate](certificate.md) | Import and manage TLS/SSL and CA certificates |
| [package](package.md) | Browse the available load balancer packages |

`loadbalancer` can be abbreviated to `lb`, and `certificate` to `ca`.

## Quick start workflow

A load balancer routes traffic through a **listener** (the public port) to a
**pool** (the backend servers). Create them in this order:

```bash
# 1. Pick a package
grn vlb package list --output table

# 2. Create the load balancer in an existing subnet
grn vlb loadbalancer create \
    --name my-lb \
    --package-id <package-id> \
    --scheme Internet \
    --type "Layer 7" \
    --subnet-id <subnet-id>

grn vlb loadbalancer wait-active --load-balancer-id <lb-id>

# 3. Create a backend pool with its health monitor and members
grn vlb pool create \
    --load-balancer-id <lb-id> \
    --name web-pool \
    --protocol HTTP \
    --algorithm ROUND_ROBIN \
    --health-monitor healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3 \
    --member ipAddress=10.0.0.4,port=80,monitorPort=80 \
    --member ipAddress=10.0.0.5,port=80,monitorPort=80

grn vlb loadbalancer wait-active --load-balancer-id <lb-id>

# 4. Create a listener that forwards to the pool
grn vlb listener create \
    --load-balancer-id <lb-id> \
    --name http-80 \
    --protocol HTTP \
    --protocol-port 80 \
    --default-pool-id <pool-id>

# 5. Check the result
grn vlb loadbalancer get --load-balancer-id <lb-id> --output table
```

You can also pass an initial listener and pool to `loadbalancer create` with
`--listener` and `--pool` to do all three in one call.

## One change at a time

Every create, update and delete puts the load balancer into `UPDATING`, and the
next change is rejected with
`400 The load balancer id ... is not ready` until it returns to `ACTIVE`.
Creating a load balancer takes a few minutes. Wait between steps with:

```bash
grn vlb loadbalancer wait-active --load-balancer-id <lb-id>
```

It exits `0` once the load balancer is `ACTIVE`, and `255` on timeout or if the
load balancer reaches `ERROR` — see
[wait-active](load-balancer.md#wait-active).

Check progress at any time with:

```bash
grn vlb loadbalancer get --load-balancer-id <lb-id> \
    --query 'data.{status:displayStatus,progress:progressStatus}'
```

## Struct-valued flags

Several flags take a small object. They accept AWS-CLI-style **shorthand** or
**JSON**:

```bash
--health-monitor healthCheckProtocol=HTTP,healthCheckPath=/,timeout=5
--health-monitor '{"healthCheckProtocol":"HTTP","healthCheckPath":"/","timeout":5}'
```

Repeatable flags (`--member`, `--rule`, `--insert-header`, `--tag`,
`--policy`, `--certificate`) are passed once per entry:

```bash
--member ipAddress=10.0.0.4,port=80 --member ipAddress=10.0.0.5,port=80
```

Because commas separate a struct's own fields, a **list nested inside shorthand
is separated with `|`** (for example `sniCertificateIds=ca-1|ca-2`). Use JSON
for anything more deeply nested.

## Previewing changes

Every command that creates or changes something accepts `--dry-run`, which
prints the exact request body without sending it. Deletions — and
`pool member update`, which replaces the whole member list — also prompt for
confirmation unless you pass `--force`.

## Global options

All VLB commands accept the global options: `--profile`, `--region`,
`--output`, `--query`, `--endpoint-url`, `--no-verify-ssl`, `--debug`,
`--color`, `--cli-read-timeout`, `--cli-connect-timeout`,
`--allow-untrusted-endpoint`. See
[Configuration](../../configuration.md).

# pool

Manage the backend pools of a load balancer, their health monitors and members.

```bash
grn vlb pool <command> [options]
grn vlb pool member <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List the pools of a load balancer |
| [get](#get) | Get details of a pool |
| [create](#create) | Create a pool |
| [update](#update) | Update a pool |
| [delete](#delete) | Delete a pool |
| [get-health-monitor](#get-health-monitor) | Get a pool's health monitor |
| [member list](#member-list) | List a pool's members |
| [member update](#member-update) | Replace a pool's members |

---

## list

List the pools of a load balancer.

### Synopsis

```
grn vlb pool list --load-balancer-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

### Examples

```bash
grn vlb pool list --load-balancer-id lb-1234abcd --output table
```

---

## get

Get details of a pool, including its health monitor and members.

### Synopsis

```
grn vlb pool get --load-balancer-id <value> --pool-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--pool-id` (string)
: Pool ID.
: - Required: Yes

---

## create

Create a backend pool. A health monitor is mandatory; members are optional and
can be added later with [member update](#member-update).

### Synopsis

```
grn vlb pool create
    --load-balancer-id <value>
    --name <value>
    --protocol <value>
    --health-monitor <value>
    [--algorithm <value>]
    [--stickiness]
    [--tls-encryption]
    [--member <value>]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--name` (string)
: Pool name. Letters, digits, `_` and `-` are allowed.
: - Required: Yes

`--protocol` (string)
: Protocol used to reach the members.
: - Required: Yes
: - Possible values: `HTTP`, `TCP`, `UDP`, `PROXY`

`--health-monitor` (struct)
: How the pool checks member health.
: - Required: Yes
: - Members: `healthCheckProtocol`, `healthCheckPath`, `healthCheckMethod`, `healthyThreshold`, `unhealthyThreshold`, `timeout`, `interval`, `successCode`, `domainName`, `httpVersion`
: - Possible values: `healthCheckProtocol` is `TCP`, `HTTP`, `HTTPS` or `PING-UDP`; `healthCheckMethod` is `GET`, `POST` or `PUT`; `httpVersion` is `1.0` or `1.1`
: - Constraints: `healthyThreshold` and `unhealthyThreshold` 2–10; `timeout` 2–120 s; `interval` 5–3600 s
: - Shorthand: `healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3`
: - JSON: `{"healthCheckProtocol":"TCP","timeout":5,"interval":30,"healthyThreshold":3,"unhealthyThreshold":3}`

!!! warning "The service requires more than the API reference suggests"
    `timeout`, `interval`, `healthyThreshold` and `unhealthyThreshold` are
    **always required** — the service returns HTTP 500, not a validation error,
    if any is missing. With `healthCheckProtocol` set to `HTTP` or `HTTPS` you
    must also give `healthCheckPath`, `healthCheckMethod`, `successCode` and
    `httpVersion`; and `httpVersion=1.1` additionally requires `domainName`.
    The CLI checks all of this before sending and tells you exactly what is
    missing.

`--algorithm` (string)
: How requests are distributed. Default: `ROUND_ROBIN`.
: - Possible values: `ROUND_ROBIN`, `LEAST_CONNECTIONS`, `SOURCE_IP`

`--stickiness` (boolean)
: Enable sticky sessions. Default: `false`.

`--tls-encryption` (boolean)
: Encrypt traffic to the members. Default: `false`.

`--member` (struct, repeatable)
: A backend server in the pool.
: - Members: `ipAddress`, `port`, `monitorPort`, `weight`, `name`, `backup`
: - Constraints: `ipAddress`, `port` and **`monitorPort`** are all required. Set `monitorPort` to the same value as `port` unless health checks target a different port.
: - Shorthand: `ipAddress=10.0.0.4,port=80,monitorPort=80,weight=1,backup=false`
: - JSON: `{"ipAddress":"10.0.0.4","port":80,"monitorPort":80,"weight":1,"backup":false}`

`--dry-run` (boolean)
: Validate the parameters and print the request body without creating anything.

### Examples

```bash
# TCP health check
grn vlb pool create --load-balancer-id lb-1234abcd \
    --name web-pool --protocol TCP --algorithm ROUND_ROBIN \
    --health-monitor healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3 \
    --member ipAddress=10.0.0.4,port=80,monitorPort=80 \
    --member ipAddress=10.0.0.5,port=80,monitorPort=80,backup=true

# HTTP health check — needs path, method, successCode and httpVersion too
grn vlb pool create --load-balancer-id lb-1234abcd \
    --name web-pool --protocol HTTP --algorithm ROUND_ROBIN \
    --health-monitor healthCheckProtocol=HTTP,healthCheckPath=/healthz,healthCheckMethod=GET,successCode=200,httpVersion=1.0,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3 \
    --member ipAddress=10.0.0.4,port=80,monitorPort=80
```

---

## update

Update a pool's algorithm, session settings or health monitor. The API requires
the whole object, so any field you do not pass is read from the current pool
and sent back unchanged. A pool's name and protocol cannot be changed after
creation, and neither can its health check protocol.

### Synopsis

```
grn vlb pool update
    --load-balancer-id <value>
    --pool-id <value>
    [--algorithm <value>]
    [--stickiness]
    [--tls-encryption]
    [--health-monitor <value>]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--pool-id` (string)
: Pool ID.
: - Required: Yes

`--health-monitor` (struct)
: Health monitor fields to change. These are **merged into** the current
  monitor, so you only pass what you want to change.
: - Members: as in [create](#create), except `healthCheckProtocol`, which cannot be changed.

!!! note "How the merge works"
    `pool get` does not return the health monitor, and the API rejects a pool
    update that does not carry a complete one. The command therefore reads the
    current monitor from the pool's health-monitor endpoint, applies your
    `--health-monitor` fields on top, and sends the result — so changing one
    field leaves the rest alone.

The other flags take the same values as [create](#create). At least one of them
must be given.

`--dry-run` (boolean)
: Print the request body without applying the update.

### Examples

```bash
grn vlb pool update --load-balancer-id lb-1234abcd --pool-id pool-1 \
    --algorithm LEAST_CONNECTIONS

grn vlb pool update --load-balancer-id lb-1234abcd --pool-id pool-1 \
    --health-monitor timeout=10,interval=60
```

---

## delete

Delete a pool. Its members and health monitor go with it.

### Synopsis

```
grn vlb pool delete --load-balancer-id <value> --pool-id <value> [--force] [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--pool-id` (string)
: Pool ID.
: - Required: Yes

`--force` (boolean)
: Skip the confirmation prompt. Default: `false`.

`--dry-run` (boolean)
: Show what would be deleted without deleting it. Default: `false`.

---

## get-health-monitor

Get the health monitor of a pool on its own.

### Synopsis

```
grn vlb pool get-health-monitor --load-balancer-id <value> --pool-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--pool-id` (string)
: Pool ID.
: - Required: Yes

### Examples

```bash
grn vlb pool get-health-monitor --load-balancer-id lb-1234abcd --pool-id pool-1 --output table
```

---

## member list

List the members of a pool, with their health status.

### Synopsis

```
grn vlb pool member list --load-balancer-id <value> --pool-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--pool-id` (string)
: Pool ID.
: - Required: Yes

### Examples

```bash
grn vlb pool member list --load-balancer-id lb-1234abcd --pool-id pool-1 --output table
```

---

## member update

Set the pool's member list.

The API replaces the list wholesale: **any current member you do not pass is
removed from the pool.** Pass `--keep-existing` to add to the current members
instead. The command prints the resulting list and asks for confirmation before
applying it.

### Synopsis

```
grn vlb pool member update
    --load-balancer-id <value>
    --pool-id <value>
    --member <value>
    [--keep-existing]
    [--force]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--pool-id` (string)
: Pool ID.
: - Required: Yes

`--member` (struct, repeatable)
: A backend server in the pool.
: - Required: Yes
: - Members: `ipAddress`, `port`, `monitorPort`, `weight`, `name`, `backup`
: - Constraints: `ipAddress`, `port` and **`monitorPort`** are all required on a new member. Members carried over by `--keep-existing` are exempt — they already carry the service's own value.
: - Shorthand: `ipAddress=10.0.0.4,port=80,monitorPort=80,weight=1,backup=false`
: - JSON: `{"ipAddress":"10.0.0.4","port":80,"monitorPort":80,"weight":1,"backup":false}`
: - Pass `--member ""` on its own to empty the pool.

`--keep-existing` (boolean)
: Keep the pool's current members and add the `--member` entries to them.
  Default: `false`.

`--force` (boolean)
: Skip the confirmation prompt. Default: `false`.

`--dry-run` (boolean)
: Show the resulting member list without applying it. Default: `false`.

### Examples

```bash
# Replace the whole list
grn vlb pool member update --load-balancer-id lb-1234abcd --pool-id pool-1 \
    --member ipAddress=10.0.0.4,port=80,monitorPort=80 \
    --member ipAddress=10.0.0.5,port=80,monitorPort=80

# Add one server, keeping the rest
grn vlb pool member update --load-balancer-id lb-1234abcd --pool-id pool-1 \
    --keep-existing --member ipAddress=10.0.0.6,port=80,monitorPort=80

# Preview first
grn vlb pool member update --load-balancer-id lb-1234abcd --pool-id pool-1 \
    --member ipAddress=10.0.0.4,port=80,monitorPort=80 --dry-run
```

---

## Global options

All commands accept the [global options](../../configuration.md): `--profile`,
`--region`, `--output`, `--query`, `--endpoint-url`, `--no-verify-ssl`,
`--debug`, `--color`, `--cli-read-timeout`, `--cli-connect-timeout`,
`--allow-untrusted-endpoint`.

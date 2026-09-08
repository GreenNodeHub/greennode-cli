# loadbalancer

Create, inspect, resize, clone and delete load balancers.

```bash
grn vlb loadbalancer <command> [options]
grn vlb lb <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List load balancers |
| [get](#get) | Get details of a load balancer |
| [create](#create) | Create a new load balancer |
| [delete](#delete) | Delete a load balancer |
| [resize](#resize) | Resize to a different package |
| [clone](#clone) | Clone an existing load balancer |
| [configure-scaling](#configure-scaling) | Configure the HA scaling group |
| [update-bandwidth](#update-bandwidth) | Update the bandwidth package |
| [get-clone-metadata](#get-clone-metadata) | Show what a clone would need |
| [list-scale-history](#list-scale-history) | List scaling events |
| [list-headers](#list-headers) | List insertable listener headers |
| [list-by-subnet](#list-by-subnet) | List the load balancers in a subnet |
| [wait-active](#wait-active) | Wait until a load balancer is ACTIVE |
| [wait-deleted](#wait-deleted) | Wait until a load balancer is gone |

---

## list

List the load balancers in your project.

The API returns every load balancer in one response — it does not paginate — so
there are no `--page` / `--page-size` options. Use `--query` to narrow or shape
the output.

### Synopsis

```
grn vlb loadbalancer list
    [--name <value>]
    [--zone-id <value>]
```

### Options

`--name` (string)
: Filter by load balancer name (substring match).

`--zone-id` (string)
: Filter by availability zone ID (exact match).

!!! note "Only these two filters are supported"
    The API also documents `ids`, `names`, `subnetIds`, `vpcIds` and `status`
    filters, but the service currently ignores them — a request carrying one
    returns the same full list as a request without it. They are therefore not
    offered as flags rather than appearing to work. Filter on anything else
    client-side with `--query`.

### Examples

```bash
grn vlb loadbalancer list --output table
grn vlb lb list --name web
grn vlb lb list --zone-id HCM03-1A --output table
grn vlb lb list --query 'listData[].{name:name,ip:address}'

# Filters the API does not support — do them client-side
grn vlb lb list --query "listData[?displayStatus=='ACTIVE'].name"
grn vlb lb list --query 'listData[:5]'
```

---

## get

Get the full details of a load balancer, including its nodes and package.

### Synopsis

```
grn vlb loadbalancer get --load-balancer-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

### Examples

```bash
grn vlb loadbalancer get --load-balancer-id lb-1234abcd --output table
```

---

## create

Create a new load balancer. Optionally create its first listener and pool in
the same call with `--listener` and `--pool`.

### Synopsis

```
grn vlb loadbalancer create
    --name <value>
    --package-id <value>
    --scheme <value>
    --subnet-id <value>
    --type <value>
    [--zone-id <value>]
    [--auto-scalable]
    [--tag <value>]
    [--listener <value>]
    [--pool <value>]
    [--dry-run]
```

### Options

`--name` (string)
: Load balancer name. Letters, digits, `_`, `-` and `.` are allowed.
: - Required: Yes

`--package-id` (string)
: Package ID. Run `grn vlb package list` to see the options.
: - Required: Yes

`--scheme` (string)
: Whether the load balancer is public or internal.
: - Required: Yes
: - Possible values: `Internet`, `Internal`

`--subnet-id` (string)
: Subnet the load balancer is placed in.
: - Required: Yes

`--type` (string)
: OSI layer the load balancer operates at.
: - Required: Yes
: - Possible values: `Layer 4`, `Layer 7`

`--zone-id` (string)
: Availability zone ID. Defaults to the region's default zone.

`--auto-scalable` (boolean)
: Enable autoscale mode. Default: `false`.

`--tag` (struct, repeatable)
: A resource tag.
: - Members: `key`, `value`
: - Shorthand: `key=env,value=prod`
: - JSON: `{"key":"env","value":"prod"}`

`--listener` (struct)
: An initial listener created with the load balancer. **Requires `--pool`** — the
  API rejects one without the other.
: - Members: `listenerName`, `listenerProtocol`, `listenerProtocolPort`, `timeoutClient`, `timeoutMember`, `timeoutConnection`, `allowedCidrs`, `blockedCidrs`, `defaultAction`, `certificateAuthorities`, `defaultCertificateAuthority`, `clientCertificate`, `insertHeaders`, `alpnProtocols`, `tlsSecurityPolicy`
: - Shorthand: `listenerName=web,listenerProtocol=HTTP,listenerProtocolPort=80`
: - JSON: `{"listenerName":"web","listenerProtocol":"HTTP","listenerProtocolPort":80}`
: - `timeoutClient`, `timeoutMember` and `timeoutConnection` are required by the
  API. The CLI defaults them to 50, 50 and 5 — the same values `listener create`
  uses — so you only set them to override.

`--pool` (struct)
: An initial pool created with the load balancer. **Requires `--listener`.**
: - Members: `poolName`, `poolProtocol`, `algorithm`, `stickiness`, `tlsEncryption`, `healthMonitor`, `members`
: - **JSON only in practice**: `healthMonitor` is required by the API and is a
  nested object, which the `key=value` shorthand cannot express (commas separate
  the struct's own fields; `|` only builds nested lists).
: - `stickiness` and `tlsEncryption` apply to an HTTP pool only; on a Layer 4
  load balancer, whose pools are `TCP`/`UDP`/`PROXY`, leave them out.
: - JSON:
  ```json
  {"poolName":"web","poolProtocol":"HTTP","algorithm":"ROUND_ROBIN",
   "stickiness":false,"tlsEncryption":false,
   "healthMonitor":{"healthCheckProtocol":"TCP","timeout":5,"interval":30,
                    "healthyThreshold":3,"unhealthyThreshold":3}}
  ```

`--dry-run` (boolean)
: Validate the parameters and print the request body without creating anything.

### Examples

```bash
grn vlb loadbalancer create \
    --name my-lb --package-id lbp-small \
    --scheme Internet --type "Layer 7" --subnet-id sub-1234

# Load balancer, listener and pool in one call
grn vlb lb create --name my-lb --package-id lbp-small \
    --scheme Internet --type "Layer 7" --subnet-id sub-1234 \
    --listener '{"listenerName":"http-80","listenerProtocol":"HTTP","listenerProtocolPort":80}' \
    --pool '{"poolName":"web","poolProtocol":"HTTP","algorithm":"ROUND_ROBIN","healthMonitor":{"healthCheckProtocol":"TCP"}}'
```

---

## delete

Delete a load balancer. Its listeners, pools and members go with it.

### Synopsis

```
grn vlb loadbalancer delete --load-balancer-id <value> [--force] [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--force` (boolean)
: Skip the confirmation prompt. Default: `false`.

`--dry-run` (boolean)
: Show what would be deleted without deleting it. Default: `false`.

### Examples

```bash
grn vlb loadbalancer delete --load-balancer-id lb-1234abcd --dry-run
grn vlb lb delete --load-balancer-id lb-1234abcd --force
```

---

## resize

Move a load balancer to a different package.

### Synopsis

```
grn vlb loadbalancer resize --load-balancer-id <value> --package-id <value> [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--package-id` (string)
: Target package ID. Run `grn vlb package list` to see the options.
: - Required: Yes

`--dry-run` (boolean)
: Print the request body without applying the resize.

### Examples

```bash
grn vlb loadbalancer resize --load-balancer-id lb-1234abcd --package-id lbp-medium
```

---

## clone

Create a new load balancer from an existing one. Run
[get-clone-metadata](#get-clone-metadata) first to see which listener
certificates the clone needs.

### Synopsis

```
grn vlb loadbalancer clone
    --load-balancer-id <value>
    --name <value>
    --package-id <value>
    [--certificate <value>]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: ID of the load balancer to clone.
: - Required: Yes

`--name` (string)
: Name of the new load balancer.
: - Required: Yes

`--package-id` (string)
: Package ID for the new load balancer.
: - Required: Yes

`--certificate` (struct, repeatable)
: Certificate override for one listener of the clone.
: - Members: `listenerId`, `defaultCertificateId`, `sniCertificateIds`
: - Shorthand: `listenerId=li-1,defaultCertificateId=ca-1,sniCertificateIds=ca-2|ca-3`
: - Constraints: in shorthand, `sniCertificateIds` is separated with `|` because commas already separate the struct's fields.

`--dry-run` (boolean)
: Print the request body without creating the clone.

### Examples

```bash
grn vlb loadbalancer get-clone-metadata --load-balancer-id lb-1234abcd
grn vlb loadbalancer clone --load-balancer-id lb-1234abcd \
    --name my-lb-copy --package-id lbp-small \
    --certificate listenerId=li-1,defaultCertificateId=ca-9
```

---

## configure-scaling

Set the instance-count bounds and the subnets the load balancer's HA group
spans.

### Synopsis

```
grn vlb loadbalancer configure-scaling
    --load-balancer-id <value>
    --max-size <value>
    --subnets <value>
    [--min-size <value>]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--max-size` (integer)
: Maximum number of instances.
: - Required: Yes
: - Constraints: must be greater than 0.

`--subnets` (string)
: All active subnet IDs, comma-separated.
: - Required: Yes
: - Constraints: must include the load balancer's primary subnet.

`--min-size` (integer)
: Minimum number of instances. Left untouched when the flag is not passed.

`--dry-run` (boolean)
: Print the request body without applying the change.

### Examples

```bash
grn vlb loadbalancer configure-scaling --load-balancer-id lb-1234abcd \
    --min-size 2 --max-size 6 --subnets sub-1234,sub-5678
```

---

## update-bandwidth

Attach a different bandwidth package to a load balancer.

### Synopsis

```
grn vlb loadbalancer update-bandwidth --load-balancer-id <value> --bandwidth-package-id <value> [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--bandwidth-package-id` (string)
: Target bandwidth package ID.
: - Required: Yes

`--dry-run` (boolean)
: Print the request body without applying the change.

### Examples

```bash
grn vlb loadbalancer update-bandwidth --load-balancer-id lb-1234abcd \
    --bandwidth-package-id bandwidth-pkg-01
```

---

## get-clone-metadata

Show the suggested name and the per-listener certificates a clone of this load
balancer would need.

### Synopsis

```
grn vlb loadbalancer get-clone-metadata --load-balancer-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

### Examples

```bash
grn vlb loadbalancer get-clone-metadata --load-balancer-id lb-1234abcd
```

---

## list-scale-history

List the scale-out and scale-in events of a load balancer.

### Synopsis

```
grn vlb loadbalancer list-scale-history --load-balancer-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

### Examples

```bash
grn vlb loadbalancer list-scale-history --load-balancer-id lb-1234abcd --output table
```

---

## list-headers

List the header names a listener can insert, for use with
`grn vlb listener create --insert-header`.

### Synopsis

```
grn vlb loadbalancer list-headers
```

### Examples

```bash
grn vlb loadbalancer list-headers
```

---

## list-by-subnet

List the load balancers attached to a subnet.

### Synopsis

```
grn vlb loadbalancer list-by-subnet --subnet-id <value>
```

### Options

`--subnet-id` (string)
: Subnet ID.
: - Required: Yes

### Examples

```bash
grn vlb loadbalancer list-by-subnet --subnet-id sub-1234 --output table
```

---

## wait-active

Poll until the load balancer reaches `ACTIVE`.

Use this between changes. Every create, update and delete puts the load balancer
into `UPDATING`, and the next change is rejected with
`400 The load balancer id ... is not ready` until it settles, so a script that
issues more than one command has to wait in between.

### Synopsis

```
grn vlb loadbalancer wait-active
    --load-balancer-id <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--delay` (integer)
: Seconds between polls. Default: `10`.

`--max-attempts` (integer)
: Give up after this many polls. Default: `90` (15 minutes at the default delay).

### Output

Progress is written to stderr so it does not pollute a pipe. On success a
confirmation goes to stdout and the exit status is `0`. If the load balancer
reaches `ERROR`, the wait times out, or the load balancer does not exist, the
command exits with status **255** — the same convention the AWS CLI uses for
waiters, so scripts can tell a failed wait from an ordinary error.

### Examples

```bash
# Provision, then wait before configuring
LB=$(grn vlb loadbalancer create --name my-lb --package-id <pkg> \
       --scheme Internet --type "Layer 7" --subnet-id <subnet> --query 'uuid' --output text)
grn vlb loadbalancer wait-active --load-balancer-id "$LB"

grn vlb pool create --load-balancer-id "$LB" --name web-pool --protocol TCP \
    --health-monitor healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3 \
    --member ipAddress=10.0.0.4,port=80,monitorPort=80
grn vlb loadbalancer wait-active --load-balancer-id "$LB"

grn vlb listener create --load-balancer-id "$LB" --name http-80 \
    --protocol HTTP --protocol-port 80
```

---

## wait-deleted

Poll until the load balancer is gone (the API returns 404).

### Synopsis

```
grn vlb loadbalancer wait-deleted
    --load-balancer-id <value>
    [--delay <value>]
    [--max-attempts <value>]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--delay` (integer)
: Seconds between polls. Default: `10`.

`--max-attempts` (integer)
: Give up after this many polls. Default: `90`.

### Output

As for [wait-active](#wait-active): exit `0` on success, exit **255** on
timeout, or if the load balancer returns to `ACTIVE` (meaning the delete never
took effect).

### Examples

```bash
grn vlb loadbalancer delete --load-balancer-id lb-1234abcd --force
grn vlb loadbalancer wait-deleted --load-balancer-id lb-1234abcd
```

---

## Global options

All commands accept the [global options](../../configuration.md): `--profile`,
`--region`, `--output`, `--query`, `--endpoint-url`, `--no-verify-ssl`,
`--debug`, `--color`, `--cli-read-timeout`, `--cli-connect-timeout`,
`--allow-untrusted-endpoint`.

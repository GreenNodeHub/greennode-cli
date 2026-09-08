# listener

Manage load balancer listeners and their L7 routing policies.

```bash
grn vlb listener <command> [options]
grn vlb listener policy <command> [options]
```

!!! note "Policy order cannot be changed from the CLI"
    The API has a reorder endpoint, but it is not exposed: the only attempt
    against the live service returned HTTP 200, changed no position, and left
    the load balancer stuck in `UPDATING` for about 30 minutes, refusing
    further changes and deletion alike. Delete and recreate policies to change
    their order, or use the console.

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List the listeners of a load balancer |
| [get](#get) | Get details of a listener |
| [create](#create) | Create a listener |
| [update](#update) | Update a listener |
| [delete](#delete) | Delete a listener |
| [policy list](#policy-list) | List the L7 policies of a listener |
| [policy get](#policy-get) | Get details of an L7 policy |
| [policy create](#policy-create) | Create an L7 policy |
| [policy update](#policy-update) | Update an L7 policy |
| [policy delete](#policy-delete) | Delete an L7 policy |

---

## list

List the listeners of a load balancer.

### Synopsis

```
grn vlb listener list --load-balancer-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

### Examples

```bash
grn vlb listener list --load-balancer-id lb-1234abcd --output table
```

---

## get

Get details of a listener.

### Synopsis

```
grn vlb listener get --load-balancer-id <value> --listener-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--listener-id` (string)
: Listener ID.
: - Required: Yes

### Examples

```bash
grn vlb listener get --load-balancer-id lb-1234abcd --listener-id li-5678
```

---

## create

Create a listener on a load balancer.

### Synopsis

```
grn vlb listener create
    --load-balancer-id <value>
    --name <value>
    --protocol <value>
    --protocol-port <value>
    [--timeout-client <value>]
    [--timeout-member <value>]
    [--timeout-connection <value>]
    [--default-pool-id <value>]
    [--allowed-cidrs <value>]
    [--blocked-cidrs <value>]
    [--default-action <value>]
    [--certificate-authorities <value>]
    [--default-certificate-authority <value>]
    [--client-certificate <value>]
    [--insert-header <value>]
    [--alpn-protocols <value>]
    [--tls-security-policy <value>]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--name` (string)
: Listener name. Letters, digits, `_` and `-` are allowed.
: - Required: Yes

`--protocol` (string)
: Listener protocol.
: - Required: Yes
: - Possible values: `HTTP`, `HTTPS`, `TCP`, `UDP`

`--protocol-port` (integer)
: Port the listener accepts traffic on.
: - Required: Yes
: - Constraints: 1–65535.

`--timeout-client` (integer)
: Client idle timeout in seconds. Default: `50`. Constraints: 1–3600.

`--timeout-member` (integer)
: Member idle timeout in seconds. Default: `50`. Constraints: 1–3600.

`--timeout-connection` (integer)
: Connection idle timeout in seconds. Default: `5`. Constraints: 1–3600.

`--default-pool-id` (string)
: Pool this listener forwards to when no L7 policy matches.

`--allowed-cidrs` (string)
: CIDRs allowed to reach the listener.

`--blocked-cidrs` (string)
: CIDRs blocked from the listener.

`--default-action` (string)
: What to do with traffic that matches no rule.
: - Possible values: `accept`, `drop`

`--certificate-authorities` (string)
: Certificate IDs, comma-separated. HTTPS only.

`--default-certificate-authority` (string)
: Default certificate ID. HTTPS only.

`--client-certificate` (string)
: TLS client authentication mode.

`--insert-header` (struct, repeatable)
: A header the listener inserts into forwarded requests.
: - Members: `headerName`, `headerValue`
: - Shorthand: `headerName=X-Forwarded-For,headerValue=true`
: - JSON: `{"headerName":"X-Forwarded-For","headerValue":"true"}`
: - Run `grn vlb loadbalancer list-headers` for the supported names.

`--alpn-protocols` (string)
: ALPN protocols, comma-separated.
: - Possible values: `http/1.1`, `http/1.0`, `h2`

`--tls-security-policy` (string)
: TLS security policy.
: - Possible values: `COMPATIBLE`, `MODERN`, `STRICT`

`--dry-run` (boolean)
: Validate the parameters and print the request body without creating anything.

### Examples

```bash
grn vlb listener create --load-balancer-id lb-1234abcd \
    --name http-80 --protocol HTTP --protocol-port 80 --default-pool-id pool-1

grn vlb listener create --load-balancer-id lb-1234abcd \
    --name https-443 --protocol HTTPS --protocol-port 443 \
    --default-certificate-authority ca-1 --alpn-protocols h2,http/1.1 \
    --tls-security-policy MODERN \
    --insert-header headerName=X-Forwarded-For,headerValue=true
```

---

## update

Update a listener. The API requires the whole object, so any field you do not
pass is read from the current listener and sent back unchanged. A listener's
name, protocol and port cannot be changed after creation.

### Synopsis

```
grn vlb listener update
    --load-balancer-id <value>
    --listener-id <value>
    [--timeout-client <value>]
    [--timeout-member <value>]
    [--timeout-connection <value>]
    [--default-pool-id <value>]
    [--allowed-cidrs <value>]
    [--blocked-cidrs <value>]
    [--default-action <value>]
    [--certificate-authorities <value>]
    [--default-certificate-authority <value>]
    [--client-certificate <value>]
    [--insert-header <value>]
    [--alpn-protocols <value>]
    [--tls-security-policy <value>]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--listener-id` (string)
: Listener ID.
: - Required: Yes

At least one of the remaining flags must be given; they take the same values as
[create](#create). `--insert-header` and `--certificate-authorities` **replace**
the current lists rather than adding to them — pass an empty value to clear one.

!!! note "TLS options on a plain HTTP listener"
    The API rejects certificate and TLS-policy fields on a listener that is not
    HTTPS. It nonetheless reports those fields on such a listener, so the
    command does not send them back unless the listener is HTTPS or you set the
    flag yourself.

`--dry-run` (boolean)
: Print the request body without applying the update.

### Examples

```bash
grn vlb listener update --load-balancer-id lb-1234abcd --listener-id li-5678 \
    --timeout-client 120

# Clear the certificate list
grn vlb listener update --load-balancer-id lb-1234abcd --listener-id li-5678 \
    --certificate-authorities ""
```

---

## delete

Delete a listener. Its L7 policies go with it.

### Synopsis

```
grn vlb listener delete --load-balancer-id <value> --listener-id <value> [--force] [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--listener-id` (string)
: Listener ID.
: - Required: Yes

`--force` (boolean)
: Skip the confirmation prompt. Default: `false`.

`--dry-run` (boolean)
: Show what would be deleted without deleting it. Default: `false`.

### Examples

```bash
grn vlb listener delete --load-balancer-id lb-1234abcd --listener-id li-5678 --dry-run
```

---

## policy list

List the L7 policies of a listener, in evaluation order. `position` starts at 1
and is assigned by the service.

### Synopsis

```
grn vlb listener policy list --load-balancer-id <value> --listener-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--listener-id` (string)
: Listener ID.
: - Required: Yes

### Examples

```bash
grn vlb listener policy list --load-balancer-id lb-1234abcd --listener-id li-5678 --output table
```

---

## policy get

Get details of an L7 policy, including its match rules.

### Synopsis

```
grn vlb listener policy get --load-balancer-id <value> --listener-id <value> --policy-id <value>
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--listener-id` (string)
: Listener ID.
: - Required: Yes

`--policy-id` (string)
: L7 policy ID.
: - Required: Yes

---

## policy create

Create an L7 policy that routes matching requests to a pool or redirects them
to a URL.

### Synopsis

```
grn vlb listener policy create
    --load-balancer-id <value>
    --listener-id <value>
    --name <value>
    --action <value>
    [--redirect-pool-id <value>]
    [--redirect-url <value>]
    --rule <value>
    [--redirect-http-code <value>]
    [--keep-query-string]
    [--dry-run]
```

### Options

`--load-balancer-id` (string)
: Load balancer ID.
: - Required: Yes

`--listener-id` (string)
: Listener ID.
: - Required: Yes

`--name` (string)
: Policy name.
: - Required: Yes

`--action` (string)
: How matching traffic is routed.
: - Required: Yes
: - Possible values: `REDIRECT_TO_POOL`, `REDIRECT_TO_URL`

`--redirect-pool-id` (string)
: Pool to forward to.
: - Required: Yes when `--action` is `REDIRECT_TO_POOL`

`--redirect-url` (string)
: URL to redirect to.
: - Required: Yes when `--action` is `REDIRECT_TO_URL`

`--redirect-http-code` (integer)
: Redirect status code. `REDIRECT_TO_URL` only.
: - Possible values: `301`, `302`

`--keep-query-string` (boolean)
: Keep the query string when redirecting. Default: `false`.

`--rule` (struct, repeatable)
: A condition the request must match. All rules must match.
: - Required: Yes — at least one. The API rejects a policy with no rules; use the listener's `--default-pool-id` and `--default-action` for traffic that matches nothing.
: - Members: `ruleType`, `compareType`, `ruleValue`
: - Possible values: `ruleType` is `PATH` or `HOST_NAME`; `compareType` is `CONTAINS`, `ENDS_WITH`, `EQUAL_TO`, `REGEX` or `STARTS_WITH`
: - Shorthand: `ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api`
: - JSON: `{"ruleType":"PATH","compareType":"STARTS_WITH","ruleValue":"/api"}`

`--dry-run` (boolean)
: Validate the parameters and print the request body without creating anything.

### Examples

```bash
grn vlb listener policy create --load-balancer-id lb-1234abcd --listener-id li-5678 \
    --name route-api --action REDIRECT_TO_POOL --redirect-pool-id pool-api \
    --rule ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api

grn vlb listener policy create --load-balancer-id lb-1234abcd --listener-id li-5678 \
    --name to-https --action REDIRECT_TO_URL --redirect-url https://example.com \
    --redirect-http-code 301 --keep-query-string \
    --rule ruleType=HOST_NAME,compareType=EQUAL_TO,ruleValue=old.example.com
```

---

## policy update

Update an L7 policy. The API requires the whole object, so any field you do not
pass is read from the current policy and sent back unchanged. `--rule`
**replaces** the whole rule list.

### Synopsis

```
grn vlb listener policy update
    --load-balancer-id <value>
    --listener-id <value>
    --policy-id <value>
    [--action <value>]
    [--redirect-pool-id <value>]
    [--redirect-url <value>]
    [--redirect-http-code <value>]
    [--keep-query-string]
    [--rule <value>]
    [--dry-run]
```

### Options

`--policy-id` (string)
: L7 policy ID.
: - Required: Yes

The other flags take the same values as [policy create](#policy-create). At
least one of them must be given. Switching `--action` to `REDIRECT_TO_URL`
requires `--redirect-url` unless the policy already has one.

### Examples

```bash
grn vlb listener policy update --load-balancer-id lb-1234abcd --listener-id li-5678 \
    --policy-id pol-1 --redirect-pool-id pool-v2

grn vlb listener policy update --load-balancer-id lb-1234abcd --listener-id li-5678 \
    --policy-id pol-1 \
    --rule ruleType=HOST_NAME,compareType=EQUAL_TO,ruleValue=api.example.com
```

---

## policy delete

Delete an L7 policy. The remaining policies are renumbered.

### Synopsis

```
grn vlb listener policy delete
    --load-balancer-id <value>
    --listener-id <value>
    --policy-id <value>
    [--force]
    [--dry-run]
```

### Options

`--policy-id` (string)
: L7 policy ID.
: - Required: Yes

`--force` (boolean)
: Skip the confirmation prompt. Default: `false`.

`--dry-run` (boolean)
: Show what would be deleted without deleting it. Default: `false`.

---

## Global options

All commands accept the [global options](../../configuration.md): `--profile`,
`--region`, `--output`, `--query`, `--endpoint-url`, `--no-verify-ssl`,
`--debug`, `--color`, `--cli-read-timeout`, `--cli-connect-timeout`,
`--allow-untrusted-endpoint`.

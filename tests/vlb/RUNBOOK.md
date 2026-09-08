# VLB CLI — live E2E runbook

Manual pre-ship verification for `grn vlb` against a real project. This covers what the Go tests
cannot: the `RunE` network path (resolve → request → output). See
[`go/cmd/vlb/CLAUDE.md`](../../go/cmd/vlb/CLAUDE.md) — every API quirk documented there was found
by running this sequence by hand.

**This is a release gate.** `grn vlb` ships behind `//go:build !vks_only`
([`go/cmd/register_vlb.go`](../../go/cmd/register_vlb.go)); do not remove that tag until this
runbook has been run end to end and `RESULTS.md` is attached to the release PR.

It is deliberately **not** a script. Every mutation puts the load balancer into `UPDATING` and the
next one fails with `400 ... is not ready`, so each step waits for `ACTIVE` before the next. Budget
**~2 hours** of wall-clock and **3–4 real load balancers**.

## Before you start

- Copy `env.example.sh` to `env.sh` and fill it in. `env.sh` is gitignored — **never** put
  credentials in it, and never commit it.
- Credentials go in the environment or in a `grn configure` profile, never in a file in this repo.
- Every resource this runbook creates is named `$PREFIX-*` and tagged `owner=$PREFIX`. Never touch
  a resource without that prefix — this runs against a shared project.
- Copy `RESULTS.template.md` to `RESULTS.md` and fill it in as you go. Redact secrets before
  pasting any output.

```bash
cp tests/vlb/env.example.sh tests/vlb/env.sh   # then edit
source tests/vlb/env.sh
cp tests/vlb/RESULTS.template.md tests/vlb/RESULTS.md
```

## Conventions used below

`$GRN` is the binary under test. `$LB`, `$POOL`, `$LN`, `$CA`, `$P1`, `$P2` hold IDs captured as
you go. After **every** mutation:

```bash
$GRN vlb lb wait-active --load-balancer-id $LB
```

**Capturing IDs from a create.** Most `POST` responses are a bare `{"<key>": "<id>"}` string map,
not a full resource — and the key name is **not in the OpenAPI spec** (it is declared only as
`additionalProperties: string`). So on the first create of each kind, print the raw response and
read the key off it, then reuse that key for the rest of the run:

```bash
$GRN vlb lb create ... --output json          # note the key, e.g. {"<key>": "lb-..."}
LB=$($GRN vlb lb create ... --query '<key>' --output text)
```

Record the actual key for each resource in `RESULTS.md` — it is undocumented behaviour worth
pinning. `certificate create` is the exception: it returns `{"data": {CADto}}`, so use
`--query 'data.uuid'`.

---

## Phase 0 — Preflight (creates nothing)

### 0.1 Build and release gate

```bash
cd go
CGO_ENABLED=0 go build -o /tmp/grn .            # NOT -tags vks_only
export GRN=/tmp/grn
$GRN vlb --help                                  # must list 5 noun groups
```

| # | Check | Expected |
|---|---|---|
| 0.1a | `$GRN vlb --help` | lists exactly `certificate listener loadbalancer package pool` (5 top-level groups; `policy` and `member` are nested) |
| 0.1b | `go build -tags vks_only -o /tmp/grn-rel . && /tmp/grn-rel vlb --help` | **fails** — `vlb` is not in the release binary |
| 0.1c | `go test ./...` | all green (baseline) |
| 0.1d | `$GRN vlb lb --help` and `$GRN vlb ca --help` | aliases resolve |
| 0.1e | `$GRN vlb loadbalancer` (bare group) | prints help, does not error on stray args |

### 0.2 Credentials

Environment variables take priority over the credentials file and never touch disk:

```bash
export GRN_CLIENT_ID=<client id>
export GRN_CLIENT_SECRET=<secret>
export GRN_DEFAULT_PROJECT_ID=<project id>
```

Or configure a profile (file created 0600, directory 0700):

```bash
$GRN configure --profile vlb-e2e
```

| # | Check | Expected |
|---|---|---|
| 0.2a | `$GRN configure list --profile vlb-e2e` | `client_id` / `client_secret` **masked** to the last 4 characters. If a full value prints, **stop the runbook** — that is a security bug |
| 0.2b | `$GRN configure get client_secret --profile vlb-e2e` | masked as well |

### 0.3 Endpoint

VLB has its own gateway, distinct from `vserver_endpoint`
([`go/internal/config/config.go`](../../go/internal/config/config.go)):

```bash
$GRN vlb package list --zone-id $ZONE --debug 2>&1 | grep -i 'vlb-gateway'
```

| # | Check | Expected |
|---|---|---|
| 0.3a | the grep above | hits `https://hcm-3.api.vngcloud.vn/vserver/vlb-gateway` |
| 0.3b | `--debug` output | contains **no** bearer token — redacted |
| 0.3c | `$GRN vlb lb list --no-verify-ssl` | warning on **stderr**, not stdout |
| 0.3d | `$GRN vlb lb list --region BOGUS` | `invalid region: BOGUS` |

### 0.4 Missing project ID

Every VLB path embeds the project ID, so an unset one must fail before any request — never build
`/v2//loadBalancers`.

```bash
env -u GRN_DEFAULT_PROJECT_ID $GRN vlb loadbalancer list --profile no-such-project
```

Expected: `project_id is not configured. Run 'grn configure' or set GRN_DEFAULT_PROJECT_ID`.

### 0.5 ID validation

Each must fail with `invalid <flag>: '<v>'. Must contain only alphanumeric characters and hyphens`
**before** any HTTP call (add `--debug` to confirm no request went out):

```bash
for bad in 'lb_bad' '-lb' 'lb-' 'a' ''; do
  $GRN vlb lb get --load-balancer-id "$bad"; echo "exit=$?"
done
```

Repeat for `--listener-id`, `--pool-id`, `--policy-id`, `--certificate-id`, and
`list-by-subnet --subnet-id`.

---

## Phase 1 — Read-only against existing resources

Run this before creating anything. It proves the three response envelopes parse against real
payloads. Pick an existing `ACTIVE` load balancer and record its ID as `$LB_REF`, plus a pool
`$POOL_REF` and listener `$LN_REF` under it.

```bash
$GRN vlb lb list --output table
LB_REF=<pick one>
```

| # | Command | Assert |
|---|---|---|
| 1.1 | `package list --zone-id $ZONE` | active packages for the zone; columns `uuid name type displayLbType connectionNumber dataTransfer mode`. Record `PKG_NLB` (an `L4`/`Network` package) and `PKG_ALB` (`L7`/`Application`) |
| 1.2 | `package list --all` | more rows than 1.1 (path is `all_packages`) |
| 1.3 | `package list --all --zone-id $ZONE` | **identical to 1.2** — `--all` silently ignores `--zone-id`. Confirm this is intended |
| 1.4 | `loadbalancer list` | envelope `listData`; `page`/`pageSize`/`totalPage` all `0` — the service ignores pagination, so `totalItem` is just the response length |
| 1.5 | `loadbalancer list --name <substring>` | genuinely narrows the result |
| 1.6 | `loadbalancer list --zone-id $ZONE` | exact-match filter works |
| 1.7 | `loadbalancer get --load-balancer-id $LB_REF` | envelope `data`; `displayStatus`, `type` (`Layer 4`/`Layer 7`), `loadBalancerSchema` present |
| 1.8 | `loadbalancer list-headers` | returns **without** `--load-balancer-id` — confirm that is intended, the path hangs off the LB collection |
| 1.9 | `loadbalancer list-by-subnet --subnet-id $SUBNET_ID` | returns a list |
| 1.10 | `loadbalancer get-clone-metadata --load-balancer-id $LB_REF` | returns metadata |
| 1.11 | `loadbalancer list-scale-history --load-balancer-id $LB_REF` | **bare array** envelope; columns `id eventType adjustmentNumber total scaledAt` |
| 1.12 | `listener list --load-balancer-id $LB_REF` | envelope `data` array. Table shows `defaultPoolId`; confirm raw JSON still has `defaultPoolName: null` even with a default pool attached |
| 1.13 | `listener get --load-balancer-id $LB_REF --listener-id $LN_REF` | timeouts, `defaultAction`, and on an HTTPS listener `tlsSecurityPolicy` + `alpnProtocols` |
| 1.14 | `listener policy list --load-balancer-id $LB_REF --listener-id $LN_REF` | on a listener that has policies: `position` values start at **1** — this is the evidence the spec's "zero-based" note is wrong |
| 1.15 | `pool list --load-balancer-id $LB_REF` | columns `uuid name protocol loadBalanceMethod stickiness tlsEncryption displayStatus` |
| 1.16 | `pool get --load-balancer-id $LB_REF --pool-id $POOL_REF` | **no `healthMonitor`, no `members`** — only scalar fields. This is why `pool update` needs a second GET |
| 1.17 | `pool get-health-monitor --load-balancer-id $LB_REF --pool-id $POOL_REF` | the monitor; this is its only source |
| 1.18 | `pool member list --load-balancer-id $LB_REF --pool-id $POOL_REF` | columns `uuid name address protocolPort weight monitorPort backup displayStatus` |
| 1.19 | `certificate list` | envelope `listData`. **No `payload` / private-key material in the output** |
| 1.20 | `certificate get --certificate-id <id>` | `algorithm`, `bitLength`, `certificateType`, `inUse` |

### 1.21 Output formats and JMESPath

Run each of these against at least one list and one get:

```bash
$GRN vlb lb list --output json
$GRN vlb lb list --output table
$GRN vlb lb list --output text
$GRN vlb lb list --query "listData[?displayStatus=='ACTIVE'].name"
$GRN vlb lb get --load-balancer-id $LB_REF --query 'data.{status:displayStatus,addr:address}'
```

Record in `RESULTS.md` per command. `--output text` on a list must not crash on nested fields.

---

## Phase 2 — NLB (Layer 4) lifecycle

Dry-run every mutation first and compare the printed body with what the real call sends
(`--debug`). Note `cli.PrintDryRun` renders maps with `%v` (`key:owner value:grn-e2e`), **not**
JSON — assert against that shape.

### 2.1 Create

```bash
$GRN vlb lb create --name $PREFIX-nlb --package-id $PKG_NLB \
  --scheme Internet --type "Layer 4" --subnet-id $SUBNET_ID --zone-id $ZONE \
  --tag key=owner,value=$PREFIX --dry-run

LB=$($GRN vlb lb create --name $PREFIX-nlb --package-id $PKG_NLB \
  --scheme Internet --type "Layer 4" --subnet-id $SUBNET_ID --zone-id $ZONE \
  --tag key=owner,value=$PREFIX --query '<lb key>' --output text)
echo "LB=$LB"
$GRN vlb lb wait-active --load-balancer-id $LB
```

Assert: `--type "Layer 4"` survives shell quoting; `wait-active` exits `0`; progress lines go to
**stderr** and the success line to **stdout**.

### 2.2 Pool create

```bash
POOL=$($GRN vlb pool create --load-balancer-id $LB --name $PREFIX-pool \
  --protocol PROXY --algorithm ROUND_ROBIN \
  --health-monitor healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3 \
  --member ipAddress=10.20.12.13,port=443,monitorPort=443,weight=1,backup=false \
  --query '<pool key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $LB
```

Then read it back: `pool get`, `pool get-health-monitor`, `pool member list` — the monitor and the
member must match what was sent.

### 2.3 Listener create

```bash
LN=$($GRN vlb listener create --load-balancer-id $LB --name $PREFIX-ln \
  --protocol TCP --protocol-port 443 --default-pool-id $POOL \
  --default-action accept --timeout-client 50 --timeout-member 50 --timeout-connection 5 \
  --query '<listener key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $LB
```

### 2.4 Listener update — PUT is not partial ⚠️ P0

`updateListener` declares all three timeouts required, so the command GETs current state, seeds the
body from it, and overlays only what you set. Change **one** field:

```bash
$GRN vlb listener update --load-balancer-id $LB --listener-id $LN --timeout-client 60 --debug
$GRN vlb lb wait-active --load-balancer-id $LB
$GRN vlb listener get --load-balancer-id $LB --listener-id $LN
```

**Assert:** `timeoutMember` and `timeoutConnection` are still `50` (not nulled, not zeroed), and
`defaultPoolId` still points at `$POOL`. This is the most dangerous bug class in VLB — a partial
update that silently wipes the fields you did not mention.

### 2.5 Listener update on TCP must not send TLS fields ⚠️ P0

The service reports TLS fields on a TCP listener as `null`/empty arrays, but rejects them on write.
Echoing its own state back therefore used to fail with `400 Cannot specify client certificate,
certificate authorities, default certificate authority, or tls security policy for http listener`.

```bash
$GRN vlb listener update --load-balancer-id $LB --listener-id $LN --timeout-client 55 --debug
```

**Assert:** the request body contains **none** of `clientCertificate`, `certificateAuthorities`,
`defaultCertificateAuthority`, `tlsSecurityPolicy`, and the call succeeds.

Then set one explicitly on the TCP listener and record what the API says — the CLI intentionally
lets the user override the skip:

```bash
$GRN vlb listener update --load-balancer-id $LB --listener-id $LN --tls-security-policy MODERN
```

### 2.6 Pool update — two GETs, non-empty monitor ⚠️ P0

`GET /pools/{id}` omits the monitor (1.16), so `pool update` must fetch it from
`/pools/{id}/healthMonitor` and merge the overrides into that. Seeding from the pool payload used
to produce an empty monitor and fail every update.

```bash
$GRN vlb pool update --load-balancer-id $LB --pool-id $POOL --health-monitor timeout=10 --debug
$GRN vlb lb wait-active --load-balancer-id $LB
$GRN vlb pool get-health-monitor --load-balancer-id $LB --pool-id $POOL
```

**Assert:** two GETs in the debug log (`/pools/{id}` and `/pools/{id}/healthMonitor`); the request
body's `healthMonitor` is **fully populated**, not empty; `interval` is still `30` and both
thresholds still `3`; `healthCheckProtocol` is **stripped** from the body (rejected on update).

Then confirm the algorithm is seeded from DTO key `loadBalanceMethod`:

```bash
$GRN vlb pool update --load-balancer-id $LB --pool-id $POOL --algorithm LEAST_CONNECTIONS
$GRN vlb lb wait-active --load-balancer-id $LB
```

### 2.7 Member update — replace vs keep-existing ⚠️ P0

`PUT .../members` replaces the whole list. Run all four cases, `--dry-run` before each apply, and
`wait-active` after each.

| # | Command | Assert |
|---|---|---|
| 2.7a | `pool member update --load-balancer-id $LB --pool-id $POOL --member ipAddress=10.20.2.13,port=443,monitorPort=443` | preview says `(1 current -> 1 new)`; afterwards the pool has **exactly one** member and `10.20.12.13` is **gone** |
| 2.7b | `... --keep-existing --member ipAddress=10.20.22.13,port=443,monitorPort=443` | pool now has 2 members; the carried-over member is exempt from the `monitorPort` requirement |
| 2.7c | `... --member ''` | body is `{"members": []}`; the pool ends up empty. This is the only way to empty a pool |
| 2.7d | `... ` with no `--member` at all | `provide at least one --member, or pass --member '' explicitly to empty the pool` |
| 2.7e | any of the above, answer `n` at the prompt | prints `Aborted.` and exits **0** — a script cannot tell "aborted" from "done" by exit status. Record as a known limitation |
| 2.7f | `... --dry-run` | still performs a GET first, so it needs credentials |

Restore one member before continuing.

### 2.8 Resize

```bash
$GRN vlb lb resize --load-balancer-id $LB --package-id <another L4 package> --dry-run
$GRN vlb lb resize --load-balancer-id $LB --package-id <another L4 package>
$GRN vlb lb wait-active --load-balancer-id $LB
$GRN vlb lb list-scale-history --load-balancer-id $LB
```

Assert a new scale event appears.

### 2.9 Clone

```bash
CLONE=$($GRN vlb lb clone --load-balancer-id $LB --name $PREFIX-nlb-clone \
  --package-id $PKG_NLB --query '<lb key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $CLONE
```

Compare `listener list` / `pool list` of the clone against the original. **Add `$CLONE` to
teardown.**

### 2.10 Configure scaling

```bash
$GRN vlb lb configure-scaling --load-balancer-id $LB \
  --min-size 1 --max-size 5 --subnets $SUBNET_ID
```

If the API rejects this on a non-autoscale LB, record that and create a third LB with
`--auto-scalable` to test it:

```bash
LB_AS=$($GRN vlb lb create --name $PREFIX-nlb-as --package-id $PKG_NLB \
  --scheme Internet --type "Layer 4" --subnet-id $SUBNET_ID --zone-id $ZONE \
  --auto-scalable --query '<lb key>' --output text)
```

### 2.11 Serialization — prove the waiter is necessary

Fire two mutations back to back with no `wait-active` between them:

```bash
$GRN vlb pool update --load-balancer-id $LB --pool-id $POOL --algorithm ROUND_ROBIN
$GRN vlb pool update --load-balancer-id $LB --pool-id $POOL --algorithm SOURCE_IP
```

Expected on the second: `400 The load balancer id ... is not ready`.

### 2.12 Teardown NLB

One delete at a time, `wait-active` between each. Dry-run first — deletes fetch state and print a
preview before the confirmation.

```bash
$GRN vlb listener delete --load-balancer-id $LB --listener-id $LN --dry-run
$GRN vlb listener delete --load-balancer-id $LB --listener-id $LN --force
$GRN vlb lb wait-active --load-balancer-id $LB

$GRN vlb pool delete --load-balancer-id $LB --pool-id $POOL --force
$GRN vlb lb wait-active --load-balancer-id $LB

$GRN vlb lb delete --load-balancer-id $LB --force
$GRN vlb lb wait-deleted --load-balancer-id $LB

$GRN vlb lb delete --load-balancer-id $CLONE --force
$GRN vlb lb wait-deleted --load-balancer-id $CLONE
```

Assert each delete preview names the resource and warns about cascading children (`listeners, pools
and members are deleted with it`; `Its L7 policies are deleted with it`).

---

## Phase 3 — ALB (Layer 7) + TLS + L7 policies

### 3.1 Local test certificate

```bash
openssl req -x509 -newkey rsa:2048 -nodes -days 30 \
  -subj "/CN=$PREFIX.example.com" -keyout /tmp/e2e.key -out /tmp/e2e.crt
```

### 3.2 Certificate dry-run — security gate

```bash
$GRN vlb certificate create --name $PREFIX-tls --type TLS/SSL \
  --certificate-file /tmp/e2e.crt --private-key-file /tmp/e2e.key --dry-run
```

**Assert:** `privateKey: <redacted>`, `passphrase: <redacted>`,
`certificate: <N bytes of PEM>`. Then grep the whole output for `BEGIN PRIVATE KEY` — it must be
**empty**. If any key material prints, stop and fix before shipping.

### 3.3 Certificate import

```bash
CA=$($GRN vlb certificate create --name $PREFIX-tls --type TLS/SSL \
  --certificate-file /tmp/e2e.crt --private-key-file /tmp/e2e.key \
  --query 'data.uuid' --output text)
$GRN vlb certificate get --certificate-id $CA
$GRN vlb certificate list --name $PREFIX
```

Also exercise the alias once with a CA-type cert:

```bash
$GRN vlb ca import --name $PREFIX-ca --type CA --certificate-file /tmp/e2e.crt
```

### 3.4 ALB create with inline listener and pool

This is a separate code path from `listener create` / `pool create`. Note the spec asymmetry:
`LbPool.members` is a single object while `CreatePoolRequest.members` is an array — record which
shape the CLI sends and whether the API accepts it.

```bash
ALB=$($GRN vlb lb create --name $PREFIX-alb --package-id $PKG_ALB \
  --scheme Internet --type "Layer 7" --subnet-id $SUBNET_ID --zone-id $ZONE \
  --listener listenerName=$PREFIX-l7,listenerProtocol=HTTP,listenerProtocolPort=80 \
  --pool poolName=$PREFIX-p7,poolProtocol=HTTP,algorithm=ROUND_ROBIN \
  --query '<lb key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $ALB
```

Also run the same command with `--dry-run` and with the JSON form
(`--listener '{"listenerName":"...","listenerProtocolPort":80}'`) to cover both parse paths.

### 3.5 HTTPS listener

```bash
POOL7=$($GRN vlb pool create --load-balancer-id $ALB --name $PREFIX-pool7 --protocol HTTP \
  --algorithm ROUND_ROBIN \
  --health-monitor healthCheckProtocol=HTTP,healthCheckPath=/,healthCheckMethod=GET,successCode=200,httpVersion=1.0,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3 \
  --member ipAddress=10.20.12.13,port=80,monitorPort=80 \
  --query '<pool key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $ALB

LN7=$($GRN vlb listener create --load-balancer-id $ALB --name $PREFIX-lns \
  --protocol HTTPS --protocol-port 443 --default-pool-id $POOL7 \
  --default-certificate-authority $CA --tls-security-policy MODERN \
  --alpn-protocols h2,http/1.1 \
  --insert-header headerName=X-Forwarded-For,headerValue=true \
  --default-action accept --debug --query '<listener key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $ALB
```

**Assert in the debug body:** `alpnProtocols` is an **array** `["h2","http/1.1"]` (comma-split),
`certificateAuthorities` would also be split, but `allowedCidrs`/`blockedCidrs` are sent as a
**raw string** (not split). Confirm that asymmetry matches the API DTO.

### 3.6 ACL and TLS policy updates

```bash
$GRN vlb listener update --load-balancer-id $ALB --listener-id $LN7 \
  --default-action drop --allowed-cidrs 0.0.0.0/0
$GRN vlb lb wait-active --load-balancer-id $ALB

$GRN vlb listener update --load-balancer-id $ALB --listener-id $LN7 --tls-security-policy STRICT
$GRN vlb lb wait-active --load-balancer-id $ALB
$GRN vlb listener get --load-balancer-id $ALB --listener-id $LN7
```

Then the asymmetric `--client-certificate` mapping — the body key is `clientCertificate` but state
is seeded from DTO key `clientCertificateAuthentication`. Round-trip it and confirm the value
survives:

```bash
$GRN vlb listener update --load-balancer-id $ALB --listener-id $LN7 --client-certificate <mode>
```

Finally clear the cert list and confirm an empty list is never echoed back:

```bash
$GRN vlb listener update --load-balancer-id $ALB --listener-id $LN7 --certificate-authorities ""
```

### 3.7 L7 policies

```bash
P1=$($GRN vlb listener policy create --load-balancer-id $ALB --listener-id $LN7 \
  --name $PREFIX-pol1 --action REDIRECT_TO_POOL --redirect-pool-id $POOL7 \
  --rule ruleType=PATH,compareType=STARTS_WITH,ruleValue=/api \
  --query '<policy key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $ALB

P2=$($GRN vlb listener policy create --load-balancer-id $ALB --listener-id $LN7 \
  --name $PREFIX-pol2 --action REDIRECT_TO_URL --redirect-url https://example.com \
  --redirect-http-code 301 --keep-query-string \
  --rule ruleType=HOST_NAME,compareType=EQUAL_TO,ruleValue=old.example.com \
  --query '<policy key>' --output text)
$GRN vlb lb wait-active --load-balancer-id $ALB
```

**`rules` is required even though the spec marks it optional** ⚠️ P0 — omitting the property
returns `400 Missing required rules property`, so the CLI always sends it:

```bash
$GRN vlb listener policy create --load-balancer-id $ALB --listener-id $LN7 \
  --name $PREFIX-pol3 --action REDIRECT_TO_POOL --redirect-pool-id $POOL7 --dry-run --debug
```

**Assert:** the body contains `rules: []`.

Read them back and record the positions the API assigned:

```bash
$GRN vlb listener policy list --load-balancer-id $ALB --listener-id $LN7
$GRN vlb listener policy get --load-balancer-id $ALB --listener-id $LN7 --policy-id $P1
```

### 3.8 Policy update — rules are replaced, not appended

```bash
$GRN vlb listener policy update --load-balancer-id $ALB --listener-id $LN7 --policy-id $P1 \
  --rule ruleType=PATH,compareType=STARTS_WITH,ruleValue=/v2 --debug
$GRN vlb lb wait-active --load-balancer-id $ALB
$GRN vlb listener policy get --load-balancer-id $ALB --listener-id $LN7 --policy-id $P1
```

**Assert:** `action` and `redirectPoolId` are seeded back from the GET (not lost), and the old
`/api` rule is **gone** — the list was replaced. Read-only DTO fields (`uuid`, `progressStatus`,
`displayStatus`) must not appear in the request body.

### 3.9 Policy reorder — REMOVED

`grn vlb listener policy update-order` **no longer exists** (`update_order.go` was deleted).
Deleting the command is the safest resolution of the `position=0` hazard, which once left a load
balancer stuck in `UPDATING` for ~30 minutes. Nothing to test here; verify the command is absent:

```bash
$GRN vlb listener policy --help   # must list only create, delete, get, list, update
```

If reordering comes back, restore the local `position >= 1` guard and this test with it.

### 3.10 Teardown ALB

Try deleting the certificate while it is still attached first — the preview must warn:

```bash
$GRN vlb certificate delete --certificate-id $CA --dry-run
```

**Assert:** `WARNING: this certificate is currently in use by a listener.`

Then unwind:

```bash
$GRN vlb listener policy delete --load-balancer-id $ALB --listener-id $LN7 --policy-id $P1 --force
$GRN vlb lb wait-active --load-balancer-id $ALB
$GRN vlb listener policy delete --load-balancer-id $ALB --listener-id $LN7 --policy-id $P2 --force
$GRN vlb lb wait-active --load-balancer-id $ALB
$GRN vlb listener delete --load-balancer-id $ALB --listener-id $LN7 --force
$GRN vlb lb wait-active --load-balancer-id $ALB
$GRN vlb pool delete --load-balancer-id $ALB --pool-id $POOL7 --force
$GRN vlb lb wait-active --load-balancer-id $ALB
$GRN vlb lb delete --load-balancer-id $ALB --force
$GRN vlb lb wait-deleted --load-balancer-id $ALB
$GRN vlb certificate delete --certificate-id $CA --force
$GRN vlb certificate delete --certificate-id $CA_CA --force
```

A policy delete preview must note that remaining policies are renumbered.

---

## Phase 4 — Validation and error paths (creates nothing)

All of these run offline or against a nonexistent ID.

### 4.1 Missing required flags

For all 37 commands, invoke with no flags and confirm cobra prints
`required flag(s) "..." not set`, naming every required flag. Script the sweep:

```bash
while read -r cmd; do
  printf '%s -> ' "$cmd"
  $GRN $cmd 2>&1 | grep -o 'required flag(s).*' || echo '(no required flags)'
done <<'EOF'
vlb loadbalancer get
vlb loadbalancer create
vlb loadbalancer delete
...
EOF
```

### 4.2 Health monitor — undocumented required fields

The spec says only `healthCheckProtocol` is required; the backend returns **HTTP 500** if the four
thresholds are missing. `pool create` validates client-side and aggregates into one message.

| # | `--health-monitor` value | Expected |
|---|---|---|
| 4.2a | `healthCheckProtocol=TCP` | `--health-monitor is missing required field(s): timeout, interval, healthyThreshold, unhealthyThreshold` |
| 4.2b | `healthCheckProtocol=HTTP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3` | `--health-monitor is missing required field(s): healthCheckPath, healthCheckMethod, successCode, httpVersion` |
| 4.2c | as 4.2b plus those four, `httpVersion=1.1` | missing `domainName` |
| 4.2d | omit `--health-monitor` entirely | cobra `required flag(s) "health-monitor" not set` |

⚠️ **`pool update` does not call `validateHealthMonitor`.** Merge a monitor that `create` would
reject and record what the API does. An HTTP 500 here is a finding to fix, not just to document.

### 4.3 Members

Exact messages, verified against this build. Use a valid `--health-monitor` so the member check is
what fails:

```bash
HM='healthCheckProtocol=TCP,timeout=5,interval=30,healthyThreshold=3,unhealthyThreshold=3'
$GRN vlb pool create --load-balancer-id lb-1 --name p --protocol TCP \
  --health-monitor "$HM" --member <case> --dry-run
```

| # | `--member` value | Expected |
|---|---|---|
| 4.3a | `ipAddress=10.0.0.4,port=80` | `--member[0]: monitorPort is required by the API (set it to the same value as port for a plain health check)` |
| 4.3b | `port=80,monitorPort=80` | `--member[0]: ipAddress is required` |
| 4.3c | `ipAddress=10.0.0.4,port=notanint,monitorPort=80` | `--member[0]: port must be an integer, got "notanint"` |
| 4.3d | a valid member, then `ipAddress=1,port=bad,monitorPort=1` | `--member[1]: port must be an integer, got "bad"` — index is correct |
| 4.3e | `ipAddress=1,port=1,monitorPort=1,backup=maybe` | `--member[0]: backup must be a boolean (true/false), got "maybe"` |

### 4.4 Enums and ranges

| # | Case | Expected |
|---|---|---|
| 4.4a | `policy create --action BOGUS` | rejected locally |
| 4.4b | `policy create --action REDIRECT_TO_POOL` with no `--redirect-pool-id` | conditional-required error |
| 4.4c | `policy create --action REDIRECT_TO_URL --redirect-http-code 303` | `--redirect-http-code must be 301 or 302, got 303` |
| 4.4d | `certificate create --type BOGUS` | `--type must be "CA" or "TLS/SSL", got "BOGUS"` |
| 4.4e | `listener create --protocol-port 70000` (and `0`) | `--protocol-port must be between 1 and 65535, got 70000` |
| 4.4f | `listener create --timeout-client 0` and `3601` | out of range 1–3600 |
| 4.4g | `configure-scaling --max-size 0` | `--max-size must be greater than 0, got 0` |
| 4.4h | `configure-scaling --subnets ""` | `--subnets must list at least one subnet ID` |
| 4.4i | `wait-active --delay 0` / `--max-attempts 0` | `must be greater than 0` |
| 4.4j | `lb create --scheme Bogus --type Bogus` | **not** validated locally (completion only) — record the API's response and decide whether to add a client-side check |

### 4.5 "Nothing to update"

Invoke each read-modify-write command with only its IDs. The message must list **every** available
body flag — a missing `--tls-security-policy` was caught this way before.

```bash
$GRN vlb listener update --load-balancer-id lb-1 --listener-id li-1
$GRN vlb pool update --load-balancer-id lb-1 --pool-id pool-1
$GRN vlb listener policy update --load-balancer-id lb-1 --listener-id li-1 --policy-id pol-1
```

### 4.6 Mutually exclusive certificate flags

All three pairs must error with cobra's `were all set`, e.g.
`if any flags in the group [certificate certificate-file] are set none of the others can be;
[certificate certificate-file] were all set`:

```bash
$GRN vlb certificate create --name x --type CA --certificate a --certificate-file b
$GRN vlb certificate create --name x --type CA --private-key a --private-key-file b
$GRN vlb certificate create --name x --type CA --certificate-chain a --certificate-chain-file b
```

Plus: no certificate at all → `a certificate is required: pass --certificate-file or
--certificate`; `--type TLS/SSL` with a cert but no key → `a private key is required for a
TLS/SSL certificate: pass --private-key-file or --private-key`; a nonexistent file →
`--certificate-file: <read error>`.

### 4.7 Struct-flag parsing

| # | Case | Expected |
|---|---|---|
| 4.7a | `--tag '{bad json'` | `invalid JSON: ...` |
| 4.7b | `--tag notakeyvalue` | `--tag[0]: invalid shorthand "notakeyvalue": expected key=value` |
| 4.7c | `--health-monitor timeout=abc` | `timeout must be an integer, got "abc"` |
| 4.7d | `--member ipAddress=1,port=1,monitorPort=1,backup=maybe` | `backup must be a boolean (true/false), got "maybe"` |
| 4.7e | `clone --certificate listenerId=li-1,defaultCertificateId=ca-1,sniCertificateIds=ca-2\|ca-3` | pipe-separated nested list parses into an array |
| 4.7f | JSON form of any struct flag | numbers arrive as numbers, not strings |

### 4.8 Not found

```bash
$GRN vlb lb get --load-balancer-id lb-00000000-0000-0000-0000-000000000000
```

Assert the API's message surfaces clearly and the CLI does not panic. Repeat for a nonexistent
listener, pool, policy, and certificate.

---

## Phase 5 — Waiters and completion

### 5.1 Waiters

`RunWaiter` calls `os.Exit` directly, so always shell out and check `$?`.

| # | Case | Expected |
|---|---|---|
| 5.1a | `wait-active` on a load balancer that is `CREATING` | progress on **stderr**, success line on **stdout**, exit **0** |
| 5.1b | `wait-active --max-attempts 1 --delay 1` on one still creating | `Waiter timed out after 1 attempts`, exit **255** |
| 5.1c | `wait-active --load-balancer-id lb-0000...` (nonexistent) | 404 is fatal — aborts immediately, exit **255** |
| 5.1d | `wait-deleted --max-attempts 2` on an `ACTIVE` load balancer | failure (the delete never took), exit **255** |
| 5.1e | `wait-deleted` right after a real delete | 404 is success, status reported `DELETED`, exit **0** |
| 5.1f | compare the waiter's status against `lb get --query 'data.displayStatus'` | the waiter reads `data.displayStatus`, not `status` |

`--delay 10 --max-attempts 90` (the defaults) is ~15 minutes of headroom.

### 5.2 Shell completion

Drive the hidden `__complete` command:

```bash
$GRN __complete vlb lb get --load-balancer-id ''
$GRN __complete vlb listener get --load-balancer-id $LB_REF --listener-id ''
$GRN __complete vlb listener get --listener-id ''          # no --load-balancer-id yet
```

| # | Case | Expected |
|---|---|---|
| 5.2a | `--load-balancer-id`, `--certificate-id`, `--package-id` | real `uuid\tname` pairs from the API |
| 5.2b | `--listener-id` / `--pool-id` / `--policy-id` with no `--load-balancer-id` on the line | empty, **no error**, shell not wedged |
| 5.2c | any API-backed completer with bad credentials | silent, no output, no hang |
| 5.2d | `--package-id` after `--zone-id $ZONE` | honours the zone |
| 5.2e | `--subnet-id` / `--zone-id` | cross-service `vserver:subnet` / `vserver:zone` |
| 5.2f | `listener create --protocol` | `HTTP HTTPS TCP UDP` — **no `PROXY`** |
| 5.2g | `pool create --protocol` | `HTTP TCP UDP PROXY` |
| 5.2h | `lb create --type` | `Layer 4` / `Layer 7` |
| 5.2i | `certificate create --type` | `CA` / `TLS/SSL` |
| 5.2j | `--certificate-file` / `--private-key-file` | file-extension filters (`pem crt cer` / `pem key`) |

Known dead entry: `--vpc-ids` is mapped to `vserver:network` in
[`go/cmd/vlb/completion.go`](../../go/cmd/vlb/completion.go) but no VLB command declares that flag.
Recommend removing it.

---

## Teardown and sign-off

```bash
$GRN vlb lb list --name $PREFIX          # must be empty
$GRN vlb certificate list --name $PREFIX # must be empty
unset GRN_CLIENT_ID GRN_CLIENT_SECRET GRN_DEFAULT_PROJECT_ID
rm -f /tmp/e2e.key /tmp/e2e.crt
```

Then rotate the credentials used for the run and note that in `RESULTS.md` (the fact only — never
the new values).

The runbook passes when:

1. `RESULTS.md` covers all 38 commands, every cell either `pass` or a recorded blocker.
2. `grep -nE 'BEGIN PRIVATE KEY|Bearer ' tests/vlb/RESULTS.md` is **empty**, and no client ID or
   secret appears anywhere in it.
3. The credentials used for the run have been rotated.
4. Both list commands above return empty — nothing leaked.
5. `go test ./...` is green, and every newly discovered quirk is (a) added to
   [`go/cmd/vlb/CLAUDE.md`](../../go/cmd/vlb/CLAUDE.md), (b) pinned by a new unit test, and
   (c) reflected in [`docs/commands/vlb/`](../../docs/commands/vlb/).

A real bug gets fixed and covered by a test before shipping — recording it here is not enough.

## Known blockers

- **`loadbalancer update-bandwidth`** needs a `bandWidthPackageId` matching
  `bandwidth-pack-<uuid>` (note the capital `W` in the API field, unlike the
  `--bandwidth-package-id` flag). No VLB command lists bandwidth packages, so there is no way to
  obtain one from the CLI. Ask the VLB team for the source. Until then this command can only be
  verified via `--dry-run` and its required-flag error — mark it **not verified live** in
  `RESULTS.md`.
- **`configure-scaling`** may only be valid on a load balancer created with `--auto-scalable`
  (see 2.10).

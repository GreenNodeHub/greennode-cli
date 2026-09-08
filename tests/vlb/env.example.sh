#!/usr/bin/env bash
# Template for the VLB live E2E runbook. Copy to env.sh and fill in.
#
#   cp tests/vlb/env.example.sh tests/vlb/env.sh
#   $EDITOR tests/vlb/env.sh
#   source tests/vlb/env.sh
#
# env.sh is gitignored. NEVER put credentials in it — this file is committed and
# its copy sits inside the repo working tree. Credentials belong in the
# environment (GRN_CLIENT_ID / GRN_CLIENT_SECRET) or in a `grn configure`
# profile under ~/.greennode (created 0600).

# Binary under test. Build WITHOUT -tags vks_only, or `grn vlb` will not exist:
#   cd go && CGO_ENABLED=0 go build -o /tmp/grn .
export GRN=/tmp/grn

# Profile to use, if you configured one instead of exporting credentials.
export PROFILE=vlb-e2e

# Availability zone to place load balancers in. Must match the subnet's zone.
export ZONE=HCM03-1C

# Subnet the load balancer VIP goes in. Find one with: grn vserver subnet list
export SUBNET_ID=

# Package IDs, from: grn vlb package list --zone-id "$ZONE"
# PKG_NLB must be an L4 / Network package, PKG_ALB an L7 / Application one.
export PKG_NLB=
export PKG_ALB=

# Prefix for every resource this run creates, and the value of its owner tag.
# Teardown greps for this, so keep it distinctive and never reuse a real name.
export PREFIX=grn-e2e

# An existing ACTIVE load balancer to read in Phase 1, plus a pool and listener
# under it. Read-only — the runbook never mutates these.
export LB_REF=
export POOL_REF=
export LN_REF=

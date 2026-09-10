# IAM

Keep management separate from existing login/logout and shared token providers. Use the global Accounts/Policies builders, explicit pagination, offline dry-runs, and no retries.

Require private regular body files for secrets. Never expose response bodies through debug/errors; secret output needs explicit opt-in. Refuse direct current-identity targets. Token issuance and impersonation remain excluded.

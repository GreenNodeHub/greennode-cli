# Configuration

## Initial setup

```bash
grn configure
```

This will prompt for:

```
GRN Client ID [None]: <your-client-id>
GRN Client Secret [None]: <your-client-secret>
Default region name [HCM-3]:
Default output format [json]:
Project ID (leave blank to auto-detect) [None]:
Fetching project_id from HCM-3...
Auto-detected project_id: pro-xxxxxxxx
```

`Project ID` is the GreenNode project UUID for the selected region (e.g.
`pro-e28d4501-...`). Leave blank and the wizard calls the vServer API with
your credentials to detect and save it. Each user has one project per region,
so the detection is unambiguous.

If auto-detect fails (network or auth error), the wizard prints a warning and
leaves the field blank — downstream tools (such as the GreenNode MCP Server)
can still auto-detect at first call.

Credentials are obtained from the [GreenNode IAM Portal](https://hcm-3.console.vngcloud.vn/iam/) under Service Accounts.

## User login (PKCE)

In addition to the machine (M2M) flow above, `grn` supports interactive user
login via a browser-based **PKCE** authorization-code flow against VNG IAM. User
login is recommended for interactive use; machine mode (`grn configure`) is
recommended for CI.

```bash
grn login                 # browser PKCE login
grn logout                # forget the cached login refresh token
```

The profile's `auth_mode` decides which credential is used at runtime:

| `auth_mode` | Source | Set by |
|---|---|---|
| `user` | PKCE refresh token | `grn login` |
| `machine` (default) | Client ID / secret | `grn configure` |

`grn login` mints a short-lived access token and persists only the **refresh
token** (plus `auth_mode` and `iam_env`) to the profile's credentials file
(`0600`). The access token is held in memory for the process only — it is never
written to disk — and is auto-refreshed before expiry (60 s skew) and again on a
`401` response. If IAM rotates the refresh token, the new one is persisted so
later invocations keep working.

### `grn login` options

| Flag | Description |
|---|---|
| `--profile <name>` | Target profile (default `default`; env `GRN_PROFILE`) |
| `--client-id <id>` | Override the baked-in public client id (env `GRN_LOGIN_CLIENT_ID`) |
| `--client-secret <secret>` | Omit for a PKCE-only public client (env `GRN_LOGIN_CLIENT_SECRET`) |
| `--scope <list>` | OAuth scopes, space-separated (default `openid`) |
| `--timeout <dur>` | Max wait for the browser flow (default `5m`) |
| `--authorize-url <url>` | Override the IAM signin/authorize URL |
| `--token-url <url>` | Override the IAM `/v2` token URL |

After the browser flow, `grn login` prompts for a default region (the same prompt
as `grn configure`), so a login-only profile is ready for `vks`/`vserver`
without a separate `grn configure`.

### Inspecting the login state

```bash
grn configure list            # shows auth_mode, iam_env, and a masked refresh_token
grn configure get auth_mode
grn configure get iam_env
```

`refresh_token` is masked in `list`/`get` output (secret-at-rest). Use
`grn logout` to clear it; machine `client_id`/`client_secret`, if present, are
left intact.

## Credential resolution order

Credentials are resolved in the following order (highest to lowest priority):

1. **Environment variables**: `GRN_ACCESS_KEY_ID`, `GRN_SECRET_ACCESS_KEY`
2. **Shared credentials file**: `~/.greennode/credentials`

## Environment variables

| Variable | Description |
|----------|-------------|
| `GRN_ACCESS_KEY_ID` | Client ID (overrides credentials file) |
| `GRN_SECRET_ACCESS_KEY` | Client Secret (overrides credentials file) |
| `GRN_DEFAULT_REGION` | Default region |
| `GRN_DEFAULT_PROJECT_ID` | Project ID (GreenNode project UUID) |
| `GRN_PROFILE` | Profile name (default: "default") |
| `GRN_DEFAULT_OUTPUT` | Output format |

Environment variables take priority over config file values.

### Example

```bash
# Set credentials via environment variables
export GRN_ACCESS_KEY_ID=your-client-id
export GRN_SECRET_ACCESS_KEY=your-client-secret
export GRN_DEFAULT_REGION=HCM-3

# Commands will use env var credentials automatically
grn vks list-clusters
```

## Config files

Credentials and config are stored in separate files:

```ini
# ~/.greennode/credentials
[default]
client_id = 5028b2cb-cb0f-4249-ae1e-1c51b2bcf6e6
client_secret = abc123

[staging]
client_id = xxx
client_secret = yyy
```

A profile created by `grn login` carries **user** identity instead of (or
alongside) machine credentials. The refresh token is stored at `0600`; the
access token is never written to disk. The four `grn login` keys land in the
same profile section and merge with any existing `client_id`/`client_secret`:

```ini
# ~/.greennode/credentials — a logged-in profile
[default]
refresh_token = rtx-xxxxxxxxxxxxxxxxxxxx
token_expires_at = 2026-08-03T18:00:00Z
auth_mode = user
iam_env = prod
# client_id / client_secret kept from a prior `grn configure` (optional)
```

```ini
# ~/.greennode/config
[default]
region = HCM-3
output = json
project_id = pro-xxxxxxxx

[profile staging]
region = HAN
output = table
project_id = pro-yyyyyyyy
```

Credentials file is created with `0600` permissions (owner read/write only).

## Configuration commands

```bash
grn configure              # Interactive setup
grn configure list         # Show all config values and sources
grn configure get region   # Get a specific value
grn configure set region HAN  # Set a specific value
```

### `grn configure list` output

```
          Name                   Value            Type    Location
          ----                   -----            ----    --------
       profile               <not set>            None    None
     client_id    ****************bc6e     config-file    ~/.greennode/credentials
 client_secret    ****************c123     config-file    ~/.greennode/credentials
        region                   HCM-3     config-file    ~/.greennode/config
        output                    json     config-file    ~/.greennode/config
    project_id       pro-xxxxxxxx          config-file    ~/.greennode/config
```

## Profiles

```bash
# Configure a named profile
grn configure --profile staging

# Log in to a named profile (user PKCE)
grn login --profile staging

# Use a named profile
grn --profile staging vks list-clusters

# Or via environment variable
export GRN_PROFILE=staging
grn vks list-clusters
```

## Available regions

| Region | VKS Endpoint |
|--------|-------------|
| `HCM-3` | `https://vks.api.vngcloud.vn` |
| `HAN` | `https://vks-han-1.api.vngcloud.vn` |

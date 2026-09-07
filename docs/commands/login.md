# grn login

Log in to GreenNode by authenticating against VNG IAM. The resulting refresh
token is folded into the per-profile `~/.greennode/credentials` (0600); the
access token is held in memory only. Use `--profile` to keep separate logins
(e.g. dev vs prod).

## Login methods

`grn login` offers two methods. When run interactively (a TTY), it shows a menu:

- **pkce** (default) — browser + local-callback PKCE authorization-code flow.
- **sso** — paste an SSO/IdP login link; the CLI runs PKCE against that authorize
  URL (with the `redirectUri` rewritten to the CLI's loopback callback).

Pick the method non-interactively with flags (see below). When stdin is not a
TTY and no method flag is given, `grn login` runs **pkce** silently.

## Browser handling

`grn login` does **not** auto-open the browser. It prints the login URL and (when
interactive) prompts:

    Press ENTER to open the browser, or 's' to skip (open the link yourself)

- **ENTER** — open the browser best-effort.
- **s** (or any other input) — skip; open the printed URL yourself on the same
  machine.
- **Ctrl-C** — cancel.

When stdin is not a TTY, the prompt is skipped and the URL is printed for you to
open manually on the same machine. (Cross-device login — opening the link on a
different machine — is not supported today; the local callback only receives
redirects from a browser on the same machine.)

While waiting for the callback, a six-dot spinner is shown on stderr so the CLI
is visibly working.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--method` | `""` | `pkce` or `sso`. Empty: menu (interactive) or pkce (non-interactive). |
| `--sso` | `""` | SSO authorize URL (non-interactive sso; implies `--method sso`). |
| `--iam-env` | `prod` | IAM environment preset: `prod` or `dev` (env: `GRN_IAM_ENV`). |
| `--client-id` | baked-in per env | IAM OAuth client id (env: `GRN_LOGIN_CLIENT_ID`). |
| `--client-secret` | `""` | Omit for a PKCE-only public client (env: `GRN_LOGIN_CLIENT_SECRET`). |
| `--authorize-url` | preset | Override the IAM signin/authorize URL (advanced). |
| `--token-url` | preset | Override the IAM `/v2` token URL (advanced). |
| `--scope` | `openid` | OAuth scopes (space-separated). |
| `--timeout` | `5m` | Maximum wait for the browser login to complete. |

## Examples

Interactive menu:

    grn login

Non-interactive pkce (CI / scripts, same machine):

    grn login

SSO with a pasted link, interactive:

    grn login --method sso

SSO non-interactive (script):

    grn login --sso https://sso.example/ap/auth?clientId=abc

Dev environment:

    grn login --iam-env dev

## Logout

    grn logout

Removes the login keys (`refresh_token`, `expiry`, `auth_mode`, `iam_env`) from
the current profile, leaving any machine `client_id`/`client_secret` intact.

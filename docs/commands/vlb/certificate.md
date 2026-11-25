# certificate

Import and manage the TLS/SSL and CA certificates used by HTTPS listeners.

```bash
grn vlb certificate <command> [options]
grn vlb ca <command> [options]
```

## Commands

| Command | Description |
|---------|-------------|
| [list](#list) | List certificates |
| [get](#get) | Get details of a certificate |
| [create](#create) | Import a certificate |
| [delete](#delete) | Delete a certificate |

---

## list

List the certificates in your project.

The API returns every certificate in one response — it does not paginate — so
there are no `--page` / `--page-size` options. Use `--query` to narrow the
output.

### Synopsis

```
grn vlb certificate list [--name <value>]
```

### Options

`--name` (string)
: Filter by certificate name (substring match).

### Examples

```bash
grn vlb certificate list --output table
grn vlb ca list --name example.com

# Expiry is not a table column — see the note below. To review it, ask for the
# field directly (notAfter is epoch milliseconds):
grn vlb certificate list --query 'listData[].{name:name,notAfter:notAfter,inUse:inUse}'
```

!!! note "The table has no expiry column"
    The service reports `expiredAt` as `null` for almost every certificate, and
    where it is set it has been seen to contradict the certificate's own
    `notAfter`. A column reading it therefore rendered blank and hid
    certificates that had already expired, so it was removed rather than left
    misleading.

    `notAfter` is the real X.509 field and is always populated, but it is epoch
    milliseconds and the table layer does no per-column formatting. Reach it
    with `--query` as above, or read it from
    [`certificate get`](#get). [`certificate delete`](#delete) formats it and
    marks `(EXPIRED)` in its confirmation preview.

---

## get

Get details of a certificate, including its subject, issuer and expiry.

### Synopsis

```
grn vlb certificate get --certificate-id <value>
```

### Options

`--certificate-id` (string)
: Certificate ID.
: - Required: Yes

---

## create

Import a certificate. Aliased as `import`.

Prefer the `--*-file` flags: passing PEM material directly on the command line
leaves your private key in the shell history. A `--dry-run` never prints the
private key or passphrase.

### Synopsis

```
grn vlb certificate create
    --name <value>
    --type <value>
    [--certificate-file <value> | --certificate <value>]
    [--private-key-file <value> | --private-key <value>]
    [--certificate-chain-file <value> | --certificate-chain <value>]
    [--passphrase <value>]
    [--dry-run]
```

### Options

`--name` (string)
: Certificate name.
: - Required: Yes

`--type` (string)
: Certificate kind.
: - Required: Yes
: - Possible values: `CA`, `TLS/SSL`

`--certificate-file` (string)
: Path to the PEM certificate.
: - Required: Yes, unless `--certificate` is given.

`--certificate` (string)
: The PEM certificate itself. Mutually exclusive with `--certificate-file`.

`--private-key-file` (string)
: Path to the PEM private key.
: - Required: Yes for a `TLS/SSL` certificate, unless `--private-key` is given.

`--private-key` (string)
: The PEM private key itself. Mutually exclusive with `--private-key-file`.

`--certificate-chain-file` (string)
: Path to the PEM certificate chain.

`--certificate-chain` (string)
: The PEM certificate chain itself. Mutually exclusive with `--certificate-chain-file`.

`--passphrase` (string)
: Passphrase protecting the private key.

`--dry-run` (boolean)
: Validate the parameters and preview the request without importing. Secret
  material is redacted from the preview.

### Examples

```bash
grn vlb certificate create --name example-com --type TLS/SSL \
    --certificate-file ./example.com.crt \
    --private-key-file ./example.com.key \
    --certificate-chain-file ./chain.pem

grn vlb ca import --name corp-root --type CA --certificate-file ./root-ca.pem
```

---

## delete

Delete a certificate. The command warns if it is currently in use by a
listener.

### Synopsis

```
grn vlb certificate delete --certificate-id <value> [--force] [--dry-run]
```

### Options

`--certificate-id` (string)
: Certificate ID.
: - Required: Yes

`--force` (boolean)
: Skip the confirmation prompt. Default: `false`.

`--dry-run` (boolean)
: Show what would be deleted without deleting it. Default: `false`.

### Examples

```bash
grn vlb certificate delete --certificate-id ca-1234 --dry-run
```

---

## Global options

All commands accept the [global options](../../configuration.md): `--profile`,
`--region`, `--output`, `--query`, `--endpoint-url`, `--no-verify-ssl`,
`--debug`, `--color`, `--cli-read-timeout`, `--cli-connect-timeout`,
`--allow-untrusted-endpoint`.

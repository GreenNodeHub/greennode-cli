# Installation

The GreenNode CLI (`grn`) is a single binary with zero dependencies. Install it with the one-liner for your platform below.

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.ps1 | iex
```

**Windows (CMD — for environments without PowerShell):**
```cmd
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/install.cmd -o install.cmd && install.cmd && del install.cmd
```

**Build from source** (fallback):
```bash
git clone https://github.com/GreenNodeHub/greennode-cli.git
cd greennode-cli/go
go build -o grn .
# place grn on your PATH
```

Verify:
```bash
grn --version
```

## Uninstall

Remove the binary + PATH entry with the one-liner for your platform:

**macOS / Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.ps1 | iex
```

**Windows (CMD — for environments without PowerShell):**
```cmd
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.cmd -o uninstall.cmd && uninstall.cmd && del uninstall.cmd
```

This removes:
- **macOS / Linux:** `~/.local/lib/greennode/`, the `~/.local/bin/grn` symlink, and the PATH line from your shell rc.
- **Windows:** `%LOCALAPPDATA%\greennode\` and its entry in the User PATH.

To also remove config + credentials (`~/.greennode/` on macOS/Linux, `%USERPROFILE%\.greennode\` on Windows), pass `--purge` (`-Purge` in PowerShell):

```bash
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.sh | bash -s -- --purge
```
```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.ps1))) -Purge
```
```cmd
curl -fsSL https://raw.githubusercontent.com/GreenNodeHub/greennode-cli/main/scripts/uninstall.cmd -o uninstall.cmd && uninstall.cmd --purge && del uninstall.cmd
```

Restart your shell (or open a new terminal) for PATH changes to take effect.

## Next steps

- Configure credentials: `grn configure`
- Enable tab completion: see [Shell Completion](usage/shell-completion.md)

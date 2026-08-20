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

## Next steps

- Configure credentials: `grn configure`
- Enable tab completion: see [Shell Completion](usage/shell-completion.md)

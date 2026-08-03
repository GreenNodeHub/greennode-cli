# Installation

## Download binary

Download the latest binary for your platform from [GitHub Releases](https://github.com/GreenNodeHub/greennode-cli/releases):

### macOS

```bash
# Apple Silicon (M1/M2/M3)
curl -L -o grn https://github.com/GreenNodeHub/greennode-cli/releases/latest/download/grn-darwin-arm64

# Intel
curl -L -o grn https://github.com/GreenNodeHub/greennode-cli/releases/latest/download/grn-darwin-amd64

chmod +x grn
sudo mv grn /usr/local/bin/
```

### Linux

```bash
# x86_64
curl -L -o grn https://github.com/GreenNodeHub/greennode-cli/releases/latest/download/grn-linux-amd64

# ARM64
curl -L -o grn https://github.com/GreenNodeHub/greennode-cli/releases/latest/download/grn-linux-arm64

chmod +x grn
sudo mv grn /usr/local/bin/
```

### Windows

Download `grn-windows-amd64.exe` from [GitHub Releases](https://github.com/GreenNodeHub/greennode-cli/releases) and add to your PATH.

## Build from source

Requires [Go 1.22+](https://go.dev/dl/):

```bash
git clone https://github.com/GreenNodeHub/greennode-cli.git
cd greennode-cli/go
go build -o grn .
sudo mv grn /usr/local/bin/
```

## Verify installation

```bash
grn --version
# grn-cli/1.5.1 Go/1.22.12 darwin/arm64
```

## Next steps

- Configure credentials: `grn configure`
- Enable tab completion: see [Shell Completion](usage/shell-completion.md)

# Shell Completion

`grn` supports command, flag, and value completion for bash, zsh, fish, and PowerShell.

## Enable

### zsh

zsh's completion system (`compinit`) must be initialized before the script is
loaded. Most frameworks (oh-my-zsh, prezto) already run `compinit`; on a vanilla
zsh you must run it yourself, otherwise you'll see `command not found: compdef`.

Current session:

```zsh
autoload -U compinit && compinit
source <(grn completion zsh)
```

Persistent — add to `~/.zshrc`:

```zsh
mkdir -p ~/.zfunc
grn completion zsh > ~/.zfunc/_grn
# then in ~/.zshrc (before any framework that calls compinit):
fpath=(~/.zfunc $fpath)
autoload -U compinit && compinit
```

### bash

```bash
source <(grn completion bash)             # current session
grn completion bash | sudo tee /etc/bash_completion.d/grn > /dev/null  # persistent
```

### fish

```bash
grn completion fish | source
grn completion fish > ~/.config/fish/completions/grn.fish  # persistent
```

### PowerShell

```powershell
grn completion powershell | Out-String | Invoke-Expression
```

## Value completion

Beyond commands and flags, `grn` completes flag *values*:

- Global: `--region`, `--output`, `--color`, `--profile`
- VKS: `--cluster-id`, `--nodegroup-id`, `--k8s-version`, `--os`, `--network-type`, `--release-channel`
- vserver resources used by VKS: `--vpc-id`, `--subnet-id`, `--subnet-ids`, `--ssh-key-id`, `--security-groups`, `--disk-type`

Resource-id completions call the API using your configured credentials and
`project_id`. They have a short timeout and fail silently — if credentials or
network are unavailable, completion simply returns nothing.

Subnet suggestions (`--subnet-ids` on `create-cluster`, `--subnet-id` on
`create-nodegroup`) need to know the VPC. They use `--vpc-id` when the command
has it, and otherwise resolve it from `--cluster-id` — so set one of the two
before asking for subnet completions. The cluster lookup costs an extra API call
inside the same short timeout, so on a slow connection completion may return
nothing where `--vpc-id` would have succeeded.

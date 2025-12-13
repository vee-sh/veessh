veessh - Console connection manager (SSH/SFTP/Telnet/Mosh/SSM)

veessh is a Go-based CLI to manage console connection profiles and credentials for
SSH, SFTP, Telnet, Mosh, AWS SSM, and other tools. It orchestrates native clients
and stores credentials securely with the system keychain.

Installation

1. Ensure Go 1.22+
2. Build:

```bash
go build -o veessh ./cmd/veessh
```

Quick start

```bash
# Add an SSH profile
./veessh add mybox --type ssh --host example.com --user alice \
  --port 22 --identity ~/.ssh/id_ed25519

# List profiles
./veessh list

# Connect
./veessh connect mybox

# Or just run veessh to get an interactive picker (like kubie ctx)
./veessh
```

Key features

- Just run `veessh` for interactive picker (like kubie ctx for k8s)
- Interactive picking with fuzzy search (built-in; uses fzf if available)
- Multiple protocols: SSH, SFTP, Telnet, Mosh, AWS SSM
- On-connect automation: remote commands, directory change, environment variables
- File transfers with `veessh scp` using profile credentials
- SSH key deployment with `veessh copy-id`
- Tmux sessions: open multiple profiles in windows/panes with `veessh session`
- Remote command execution without interactive shell (`veessh run`)
- Connectivity testing and diagnostics (`veessh test`, `veessh doctor`)
- Connection audit logging with `veessh audit`
- Port-forward presets with toggle at connect time (`--forward` / `--no-forward`)
- Favorites and recents; usage tracking updates on successful connect
- ProxyJump support; tag support; JSON output for list/show/history/audit
- Import/export profiles (YAML), and import from OpenSSH config
- Shell completions for bash/zsh/fish/powershell
- Graceful Ctrl+C: clean cancellation with "ok. exiting"

Notes

- veessh uses your system's native tools (ssh, sftp, telnet). Ensure they are
  installed and in PATH.
- Passwords are optional; when provided, they are stored in the OS keychain via
  github.com/99designs/keyring. The current version does not auto-inject
  passwords into ssh; you'll be prompted by the tool as usual. Keys and agents
  are fully supported.
- Config lives at ~/.config/veessh/config.yaml by default.

Core commands

- add: Create a new profile (ssh, sftp, telnet, mosh, ssm).
- edit: Modify an existing profile.
- clone: Duplicate a profile with a new name.
- list: Show profiles (supports --tag and --json).
- show: Show details for a profile (supports --json).
- connect: Connect using a profile (supports --forward / --no-forward).
- run: Execute a remote command without interactive shell.
- scp: Copy files to/from remote using profile credentials.
- copy-id: Deploy SSH public key to remote host.
- session: Open multiple profiles in tmux windows/panes.
- test: Check if a host is reachable.
- pick: Interactively pick and connect (supports --fzf, --favorites, --tag,
  --recent-first, --print).
- favorite: Toggle favorite flag.
- history: View recent connections and usage statistics.
- audit: View connection audit log.
- doctor: Diagnose connection issues and validate setup.
- export / import: Export/import profiles (YAML; no passwords).
- import-ssh: Import from ~/.ssh/config.
- completion: Emit shell completion script.
- remove: Delete a profile (and optionally its stored password).

Examples

SFTP and Telnet:

```bash
./veessh add filesvc --type sftp --host files.example --user alice
./veessh connect filesvc

./veessh add legacy --type telnet --host legacy.example --port 23
./veessh connect legacy
```

Picker, favorites, tags:

```bash
./veessh favorite mybox
./veessh pick --favorites --tag prod --recent-first --fzf
```

JSON output and tags on list/show:

```bash
./veessh list --tag prod --json
./veessh show mybox --json
```

Import/export and OpenSSH import:

```bash
./veessh export --file profiles.yaml
./veessh import --file profiles.yaml --overwrite
./veessh import-ssh --file ~/.ssh/config --group imported --prefix ssh-
```

Edit and clone:

```bash
# Edit an existing profile
./veessh edit mybox --host newhost.example.com --port 2222

# Clone a profile
./veessh clone prod-server staging-server --host staging.example.com
```

Remote commands:

```bash
# Run a command on remote host
./veessh run mybox uptime
./veessh run mybox "df -h"
./veessh run mybox --tty top   # Force TTY for interactive commands
```

Testing and diagnostics:

```bash
# Test connectivity
./veessh test mybox            # Single profile
./veessh test --all            # All profiles

# Diagnose issues
./veessh doctor                # Check all profiles
./veessh doctor mybox -v       # Verbose check on single profile
```

History and statistics:

```bash
./veessh history               # Recent connections
./veessh history -n 5          # Last 5 connections
./veessh history --stats       # Usage statistics
./veessh history --json        # JSON output
```

File transfers and key deployment:

```bash
# Copy files using profile credentials
./veessh scp mybox:/var/log/app.log ./app.log
./veessh scp ./config.yaml mybox:/app/config.yaml
./veessh scp -r mybox:/backup/ ./local-backup/

# Deploy SSH public key
./veessh copy-id mybox
./veessh copy-id mybox --key ~/.ssh/mykey.pub
```

Tmux sessions:

```bash
# Open multiple profiles in tmux windows
./veessh session web-server db-server cache-server

# Named session
./veessh session prod-web prod-db --name prod-cluster

# Open as panes with layout
./veessh session web1 web2 web3 --layout tiled
./veessh session master worker --layout even-horizontal
```

On-connect automation:

```bash
# Auto-attach to tmux on connect
./veessh add dev --host dev.example --user dev --remote-cmd "tmux attach || tmux new"

# Auto-cd to directory
./veessh add web --host web.example --user deploy --remote-dir /var/www/app
```

Mosh (mobile shell):

```bash
# Add mosh profile for unstable connections
./veessh add flaky --type mosh --host unstable.example --user alice
./veessh connect flaky
```

AWS SSM:

```bash
# Add SSM profile (no open ports needed)
./veessh add ec2-prod --type ssm --instance-id i-1234567890abcdef0 --aws-region us-east-1
./veessh connect ec2-prod
```

Port forwarding:

```bash
# Add profile with port forwards
./veessh add tunnel --host jump.example --user admin \
  --local-forward 8080:internal:80 --dynamic-forward 1080

# Connect with/without forwarding
./veessh connect tunnel              # With forwards
./veessh connect tunnel --no-forward # Skip forwards this time
```

Audit log:

```bash
./veessh audit           # View connection audit log
./veessh audit -n 10     # Last 10 entries
./veessh audit --json    # JSON output
```

Completions:

```bash
# Bash
./veessh completion bash > /usr/local/etc/bash_completion.d/veessh

# Zsh
./veessh completion zsh > "${fpath[1]}/_veessh"

# Fish
./veessh completion fish > ~/.config/fish/completions/veessh.fish

# PowerShell
./veessh completion powershell > veessh.ps1
```

Roadmap

- Profile templates/inheritance and shared read-only org directory merge
- Rich TUI picker with columns and quick actions (connect, edit, copy, favorite)
- Host key trust: strict/lenient modes, first-connect fingerprint verify, pinning
- Secrets integration: 1Password/Bitwarden/AWS Secrets Manager fetch
- Additional transports: GCP gcloud SSH, serial, RDP stubs
- Proxy support: SOCKS/HTTP, ProxyCommand, multi-hop chains with saved hops
- rsync integration for efficient directory sync
- Packaging: Homebrew tap, static builds

Pluggability

Connectors are pluggable; you can add new protocol handlers in
internal/connectors.

License

Apache-2.0

Releases via GitHub Actions

- Tag a version and push the tag; the release workflow builds artifacts for
  Linux and macOS (amd64/arm64) and publishes a release with tarballs and
  checksums.

```bash
git tag v0.1.1
git push origin v0.1.1
```

The workflow is defined at `.github/workflows/release.yml`.



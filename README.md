veessh - Console connection manager (SSH/SFTP/Telnet/Mosh/SSM/GCloud)

veessh is a Go-based CLI to manage console connection profiles and credentials for
SSH, SFTP, Telnet, Mosh, AWS SSM, GCP gcloud, and other tools. It orchestrates
native clients and stores credentials securely with the system keychain.

Installation

Homebrew (macOS/Linux):

```bash
brew install vee-sh/tap/veessh
```

From source (Go 1.22+):

```bash
go install github.com/vee-sh/veessh/cmd/veessh@latest
```

Or build manually:

```bash
git clone https://github.com/vee-sh/veessh.git
cd veessh
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
- Multiple protocols: SSH, SFTP, Telnet, Mosh, AWS SSM, GCP gcloud
- Profile inheritance: create templates and extend them
- On-connect automation: remote commands, directory change, environment variables
- File transfers with `veessh scp` and `veessh rsync`
- SSH key deployment with `veessh copy-id`
- Host key pinning and verification with `veessh hostkey`
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
- **TUI onboarding wizard**: First-time users get an interactive setup wizard
- **Config editor**: `veessh edit-config` opens config in your editor (vi/vim/nano/etc)
- **1Password integration**: Store passwords in 1Password instead of system keychain

Notes

- veessh uses your system's native tools (ssh, sftp, telnet). Ensure they are
  installed and in PATH.
- **Password storage**: Passwords are stored securely using:
  - **1Password** (if `op` CLI is installed and signed in) - automatically detected
  - **System keychain** (macOS Keychain, Linux Secret Service, Windows Credential Manager) - fallback
  - Set `VEESSH_CREDENTIALS_BACKEND=1password` or `VEESSH_CREDENTIALS_BACKEND=keyring` to force a specific backend
- **SSH keys**: Private keys are stored on disk (typically `~/.ssh/`). Ensure proper permissions (600) and consider using SSH agent for added security.
- Config lives at `~/.config/veessh/config.yaml` by default.

Core commands

- add: Create a new profile (ssh, sftp, telnet, mosh, ssm, gcloud).
- edit: Modify an existing profile.
- clone: Duplicate a profile with a new name.
- list: Show profiles (supports --tag and --json).
- show: Show details for a profile (supports --json).
- connect: Connect using a profile (supports --forward / --no-forward).
- run: Execute a remote command without interactive shell.
- scp: Copy files to/from remote using profile credentials.
- rsync: Efficiently sync directories with remote host.
- copy-id: Deploy SSH public key to remote host.
- session: Open multiple profiles in tmux windows/panes.
- test: Check if a host is reachable.
- pick: Interactively pick and connect (supports --fzf, --favorites, --tag,
  --recent-first, --print).
- favorite: Toggle favorite flag.
- history: View recent connections and usage statistics.
- audit: View connection audit log.
- hostkey: Manage host key verification (show, pin, verify, list).
- doctor: Diagnose connection issues and validate setup.
- export / import: Export/import profiles (YAML; no passwords).
- import-ssh: Import from ~/.ssh/config.
- edit-config: Open config file in your default editor (respects `$EDITOR`).
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

# Efficient directory sync with rsync
./veessh rsync ./dist/ mybox:/var/www/html/
./veessh rsync --delete ./dist/ mybox:/var/www/html/  # Remove extra files
./veessh rsync --dry-run ./dist/ mybox:/var/www/html/ # Preview changes

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

GCP gcloud:

```bash
# Add GCP Compute Engine profile
./veessh add gce-web --type gcloud --host my-vm --gcp-project myproject --gcp-zone us-central1-a

# Use IAP tunnel (no public IP needed)
./veessh add gce-private --type gcloud --host internal-vm --gcp-project myproject --gcp-zone us-east1-b --gcp-tunnel
```

Onboarding and configuration:

```bash
# First-time setup: run veessh with no profiles to launch interactive wizard
./veessh

# Edit config file directly in your editor
./veessh edit-config                    # Uses $EDITOR or defaults to vi
EDITOR=nano ./veessh edit-config        # Use specific editor
EDITOR=code ./veessh edit-config        # Use VS Code
```

1Password integration:

```bash
# 1Password is automatically used if 'op' CLI is installed and signed in
# Passwords are stored as "veessh - <profile-name>" items in 1Password

# Force 1Password backend
export VEESSH_CREDENTIALS_BACKEND=1password
./veessh add mybox --host example.com --user alice --ask-password

# Force system keyring backend
export VEESSH_CREDENTIALS_BACKEND=keyring
./veessh add mybox --host example.com --user alice --ask-password

# Auto-detect (default: prefers 1Password if available, falls back to keyring)
export VEESSH_CREDENTIALS_BACKEND=auto
```

Profile inheritance (templates):

```bash
# Create a template profile
./veessh add prod-template --host example.com --user deploy --identity ~/.ssh/deploy_key

# Create profiles that inherit from template
./veessh add prod-web --extends prod-template --host web.example.com
./veessh add prod-api --extends prod-template --host api.example.com
./veessh add prod-db --extends prod-template --host db.example.com --user dbadmin
```

Host key verification:

```bash
# Show a host's current fingerprint
./veessh hostkey show mybox

# Pin a host's key for future verification
./veessh hostkey pin mybox

# Verify a host's key against pinned fingerprint
./veessh hostkey verify mybox

# List all pinned keys
./veessh hostkey list
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

Security

**Password Storage:**
- Passwords are stored securely using your system's keychain or 1Password
- 1Password integration: Automatically detected if `op` CLI is installed and signed in
- System keyring: Falls back to macOS Keychain, Linux Secret Service, or Windows Credential Manager
- Passwords are never stored in plain text or in the config file

**Password Usage:**
- Stored passwords are automatically used for SSH connections when `sshpass` is installed
- Without `sshpass`, SSH will prompt for the password (password is still stored for future use)
- Install `sshpass` for automatic password injection:
  - macOS: `brew install hudochenkov/sshpass/sshpass`
  - Linux: `sudo apt-get install sshpass` (Debian/Ubuntu) or `sudo yum install sshpass` (RHEL/CentOS)

**SSH Keys:**
- Private keys remain on disk (typically `~/.ssh/`)
- Ensure proper file permissions: `chmod 600 ~/.ssh/id_*`
- Consider using SSH agent (`ssh-add`) for added security
- Keys are never stored by veessh; only paths are referenced

**Config File:**
- Config file permissions: `~/.config/veessh/config.yaml` (mode 600)
- Passwords are never included in config or exports
- Use `veessh export` to share profiles without credentials

Roadmap

- Rich TUI picker with columns and quick actions
- Additional secrets backends: Bitwarden, AWS Secrets Manager
- Additional transports: serial, RDP
- Advanced proxy: SOCKS/HTTP, ProxyCommand, multi-hop chains

Contributing

Connectors are pluggable — add new protocol handlers in `internal/connectors/`.

License

Apache-2.0 — see [LICENSE](LICENSE)

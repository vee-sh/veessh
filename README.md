veessh - Console connection manager (SSH/SFTP/Telnet)

veessh is a Go-based CLI to manage console connection profiles and credentials for
SSH, SFTP, Telnet, and other tools. It orchestrates native clients (ssh, sftp,
telnet) and stores credentials securely with the system keychain.

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
- Favorites and recents; usage tracking updates on successful connect
- Remote command execution without interactive shell (`veessh run`)
- Connectivity testing and diagnostics (`veessh test`, `veessh doctor`)
- ProxyJump support; tag support; JSON output for list/show/history
- Port-forward presets on profiles (local/remote/dynamic)
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

- add: Create a new profile.
- edit: Modify an existing profile.
- clone: Duplicate a profile with a new name.
- list: Show profiles (supports --tag and --json).
- show: Show details for a profile (supports --json).
- connect: Connect using a profile.
- run: Execute a remote command without interactive shell.
- test: Check if a host is reachable.
- pick: Interactively pick and connect (supports --fzf, --favorites, --tag,
  --recent-first, --print).
- favorite: Toggle favorite flag.
- history: View recent connections and usage statistics.
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

- On-connect automation: per-profile commands (tmux attach/new, cd, env vars),
  remote working dir
- Port-forward presets UX: toggle at connect time; named presets per profile
- Profile templates/inheritance and shared read-only org directory merge
- Rich TUI picker with columns and quick actions (connect, edit, copy, favorite)
- Host key trust: strict/lenient modes, first-connect fingerprint verify, pinning
- Secrets integration: 1Password/Bitwarden/AWS Secrets Manager fetch
- Additional transports: AWS SSM, GCP gcloud SSH, mosh, serial, RDP stubs
- Proxy support: SOCKS/HTTP, ProxyCommand, multi-hop chains with saved hops
- Teams/audit: connection audit log (timestamp/duration, no secrets), reports
- Scripting: JSON for all commands, stable schema, machine-mode to avoid prompts
- Packaging: Homebrew tap, release artifacts, static builds, CI lint/test/build
- Sessions: open multiple profiles into tmux windows, named sessions, layouts
- SCP/rsync integration: file transfer shortcuts using profile credentials
- Copy-id: ssh-copy-id integration for easy key deployment

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



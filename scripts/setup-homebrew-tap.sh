#!/bin/bash
# Setup script for creating the homebrew-tap repository
#
# This script helps you create the initial homebrew-tap repository
# that the release workflow will automatically update.
#
# Prerequisites:
# - GitHub CLI (gh) installed and authenticated
# - Write access to create repositories in your GitHub account/org
#
# Usage:
#   ./scripts/setup-homebrew-tap.sh

set -euo pipefail

# Get the authenticated GitHub username
OWNER="${1:-$(gh api user -q .login)}"
REPO="homebrew-tap"

echo "=== Homebrew Tap Setup ==="
echo ""
echo "This will create: github.com/${OWNER}/${REPO}"
echo ""

# Check if gh is available
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed."
    echo "Install it with: brew install gh"
    exit 1
fi

# Check authentication
if ! gh auth status &> /dev/null; then
    echo "Error: Not authenticated with GitHub CLI."
    echo "Run: gh auth login"
    exit 1
fi

# Create temp directory
TMPDIR=$(mktemp -d)
cd "$TMPDIR"

echo "Creating repository structure..."

# Initialize repo
git init homebrew-tap
cd homebrew-tap

# Create Formula directory
mkdir -p Formula

# Create initial formula (placeholder)
cat > Formula/veessh.rb << EOF
# typed: false
# frozen_string_literal: true

class Veessh < Formula
  desc "Console connection manager for SSH/SFTP/Telnet/Mosh/SSM/GCloud"
  homepage "https://github.com/${OWNER}/veessh"
  version "0.3.0"
  license "Apache-2.0"

  on_macos do
    on_intel do
      url "https://github.com/${OWNER}/veessh/releases/download/v#{version}/veessh_v#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_WILL_BE_UPDATED_BY_RELEASE_WORKFLOW"
    end

    on_arm do
      url "https://github.com/${OWNER}/veessh/releases/download/v#{version}/veessh_v#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_WILL_BE_UPDATED_BY_RELEASE_WORKFLOW"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/${OWNER}/veessh/releases/download/v#{version}/veessh_v#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_WILL_BE_UPDATED_BY_RELEASE_WORKFLOW"
    end

    on_arm do
      url "https://github.com/${OWNER}/veessh/releases/download/v#{version}/veessh_v#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_WILL_BE_UPDATED_BY_RELEASE_WORKFLOW"
    end
  end

  def install
    bin.install "veessh"
    generate_completions_from_executable(bin/"veessh", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/veessh --version")
  end
end
EOF

# Create README
cat > README.md << EOF
# Homebrew Tap for veessh

This tap contains the Homebrew formula for [veessh](https://github.com/${OWNER}/veessh).

## Installation

\`\`\`bash
brew tap ${OWNER}/tap
brew install veessh
\`\`\`

## Updating

\`\`\`bash
brew update
brew upgrade veessh
\`\`\`

## About veessh

veessh is a console connection manager supporting SSH, SFTP, Telnet, Mosh, AWS SSM, and GCP gcloud.

For more information, see the [main repository](https://github.com/${OWNER}/veessh).
EOF

# Commit
git add .
git commit -m "Initial formula for veessh"

echo ""
echo "Creating GitHub repository..."

# Create repo on GitHub
gh repo create "${OWNER}/${REPO}" \
    --public \
    --description "Homebrew tap for veessh" \
    --source . \
    --push

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Repository created: https://github.com/${OWNER}/${REPO}"
echo ""
echo "Next steps:"
echo ""
echo "1. Create a Personal Access Token (PAT) with 'repo' scope:"
echo "   https://github.com/settings/tokens/new?scopes=repo"
echo ""
echo "2. Add the token as a secret in your veessh repository:"
echo "   https://github.com/${OWNER}/veessh/settings/secrets/actions/new"
echo "   Name: TAP_GITHUB_TOKEN"
echo "   Value: <your PAT>"
echo ""
echo "3. Update .github/workflows/release.yml to use your tap:"
echo "   Change 'alex-vee-sh/homebrew-tap' to '${OWNER}/homebrew-tap'"
echo ""
echo "4. Push a new tag to trigger the release workflow:"
echo "   git tag -a v0.3.1 -m 'Trigger tap update'"
echo "   git push origin v0.3.1"
echo ""
echo "Users can now install with:"
echo "   brew tap ${OWNER}/tap"
echo "   brew install veessh"

# Cleanup
cd /
rm -rf "$TMPDIR"


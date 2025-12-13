#!/usr/bin/env bash
set -euo pipefail

# Usage: VERSION=v0.1.0 ./scripts/release.sh
# Output: dist/veessh_${VERSION}_${GOOS}_${GOARCH}.tar.gz + .sha256

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
CMD_PATH="${ROOT_DIR}/cmd/veessh"
APP_NAME="veessh"

VERSION="${VERSION:-v0.1.0}"
COMMIT="$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo none)"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p "${DIST_DIR}"

build_one() {
	local os="$1" arch="$2"
	local out_name="${APP_NAME}_${VERSION}_${os}_${arch}"
	local out_dir="${DIST_DIR}/${out_name}"
	mkdir -p "${out_dir}"
	local ldflags="-s -w -X github.com/vee-sh/veessh/internal/version.Version=${VERSION} -X github.com/vee-sh/veessh/internal/version.Commit=${COMMIT} -X github.com/vee-sh/veessh/internal/version.Date=${DATE}"
	GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
		go build -trimpath -ldflags "$ldflags" -o "${out_dir}/${APP_NAME}" "${CMD_PATH}"
	(
		cd "${DIST_DIR}"
		tar -czf "${out_name}.tar.gz" "${out_name}"
		shasum -a 256 "${out_name}.tar.gz" | awk '{print $1"  "$2}' > "${out_name}.tar.gz.sha256"
	)
	rm -rf "${out_dir}"
}

build_one darwin amd64
build_one darwin arm64
build_one linux amd64
build_one linux arm64

echo "Built artifacts in ${DIST_DIR}:"
ls -1 "${DIST_DIR}" | sed 's/^/  /'

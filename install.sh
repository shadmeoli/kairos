#!/usr/bin/env sh
# Install kairos into your Go toolchain bin directory (same as: go install .).
# The binary is typically $GOBIN/kairos, or $(go env GOPATH)/bin/kairos if GOBIN is empty.
# Ensure that directory is on your PATH.

set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd -- "$ROOT"

if ! command -v go >/dev/null 2>&1; then
	echo "install.sh: go is not on PATH" >&2
	exit 1
fi

go install .

bin_dir="$(go env GOBIN)"
if [ -z "$bin_dir" ]; then
	bin_dir="$(go env GOPATH)/bin"
fi

echo "Installed: $bin_dir/kairos"
echo "If \`kairos\` is not found, add to PATH\nexport PATH=\"$bin_dir:\$PATH\"\nyou can also alias it as ks to be the short hand\nalias ks=kairos"

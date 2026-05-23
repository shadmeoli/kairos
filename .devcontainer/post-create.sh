#!/usr/bin/env bash

set -euo pipefail

cd /workspaces/kairos
go mod download

ZSHRC="${HOME}/.zshrc"

if [[ -f "${ZSHRC}" ]]; then
  sed -i 's/^ZSH_THEME=.*/ZSH_THEME="robbyrussell"/' "${ZSHRC}"

  if grep -q '^plugins=' "${ZSHRC}"; then
    sed -i 's/^plugins=.*/plugins=(git golang)/' "${ZSHRC}"
  else
    printf '\nplugins=(git golang)\n' >> "${ZSHRC}"
  fi
fi

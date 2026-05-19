#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/ntts9990/ntts-flightlog.git}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

git clone --depth 1 "${REPO_URL}" "${TMP_DIR}/ntts-flightlog"
"${TMP_DIR}/ntts-flightlog/scripts/install.sh"

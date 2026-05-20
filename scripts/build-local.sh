#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out_dir="${NTTS_FLIGHTLOG_LOCAL_DIST:-${repo_root}/dist/local-current}"
out="${out_dir}/ntts-flightlog"

mkdir -p "${out_dir}"
go build -ldflags "-X main.version=local" -o "${out}" "${repo_root}/cmd/flightlog"
chmod +x "${out}"

printf '%s\n' "${out}"

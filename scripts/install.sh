#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CODEX_SKILLS_DIR="${CODEX_SKILLS_DIR:-${HOME}/.codex/skills}"
LOCAL_BIN_DIR="${LOCAL_BIN_DIR:-${HOME}/.local/bin}"

mkdir -p "${CODEX_SKILLS_DIR}" "${LOCAL_BIN_DIR}"

rm -rf "${CODEX_SKILLS_DIR}/ntts-flightlog"
cp -R "${REPO_ROOT}/skill/ntts-flightlog" "${CODEX_SKILLS_DIR}/ntts-flightlog"

cp "${REPO_ROOT}/bin/ntts-flightlog" "${LOCAL_BIN_DIR}/ntts-flightlog"
chmod +x "${LOCAL_BIN_DIR}/ntts-flightlog"

printf 'Installed Codex skill: %s\n' "${CODEX_SKILLS_DIR}/ntts-flightlog"
printf 'Installed CLI: %s\n' "${LOCAL_BIN_DIR}/ntts-flightlog"
printf 'Restart Codex to reload the skill list.\n'

#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
  cat <<'USAGE'
Usage: install.sh [options]

Options:
  --codex          Install only the Codex skill (under ~/.codex/skills).
  --claude         Install only the Claude Code skill (under ~/.claude/skills).
  --gemini         Install only the Gemini CLI skill (under ~/.gemini/skills).
  --all            Force-install to every supported agent skill directory.
  --no-cli         Skip the ~/.local/bin/ntts-flightlog CLI install.
  -h, --help       Show this message.

Default behavior (no flags):
  - Always install the standalone CLI to ~/.local/bin/ntts-flightlog.
  - Install the skill into every supported agent directory that already exists
    (so a Claude-only or Codex-only machine just gets one copy).
  - If no agent directories exist yet, create ~/.codex/skills and install there
    (matches the original behavior).

Environment overrides:
  CODEX_SKILLS_DIR    default: $HOME/.codex/skills
  CLAUDE_SKILLS_DIR   default: $HOME/.claude/skills
  GEMINI_SKILLS_DIR   default: $HOME/.gemini/skills
  LOCAL_BIN_DIR       default: $HOME/.local/bin
USAGE
}

INSTALL_CODEX=0
INSTALL_CLAUDE=0
INSTALL_GEMINI=0
INSTALL_CLI=1
EXPLICIT_TARGETS=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --codex)   INSTALL_CODEX=1;  EXPLICIT_TARGETS=1; shift ;;
    --claude)  INSTALL_CLAUDE=1; EXPLICIT_TARGETS=1; shift ;;
    --gemini)  INSTALL_GEMINI=1; EXPLICIT_TARGETS=1; shift ;;
    --all)     INSTALL_CODEX=1; INSTALL_CLAUDE=1; INSTALL_GEMINI=1; EXPLICIT_TARGETS=1; shift ;;
    --no-cli)  INSTALL_CLI=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) printf 'Unknown option: %s\n\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
done

CODEX_SKILLS_DIR="${CODEX_SKILLS_DIR:-${HOME}/.codex/skills}"
CLAUDE_SKILLS_DIR="${CLAUDE_SKILLS_DIR:-${HOME}/.claude/skills}"
GEMINI_SKILLS_DIR="${GEMINI_SKILLS_DIR:-${HOME}/.gemini/skills}"
LOCAL_BIN_DIR="${LOCAL_BIN_DIR:-${HOME}/.local/bin}"

# Auto-detect targets when none were given explicitly.
if (( ! EXPLICIT_TARGETS )); then
  [[ -d "${CODEX_SKILLS_DIR}"  || -d "${HOME}/.codex"  ]] && INSTALL_CODEX=1  || true
  [[ -d "${CLAUDE_SKILLS_DIR}" || -d "${HOME}/.claude" ]] && INSTALL_CLAUDE=1 || true
  [[ -d "${GEMINI_SKILLS_DIR}" || -d "${HOME}/.gemini" ]] && INSTALL_GEMINI=1 || true
  if (( ! INSTALL_CODEX && ! INSTALL_CLAUDE && ! INSTALL_GEMINI )); then
    # No agent detected — fall back to Codex (original behavior).
    INSTALL_CODEX=1
  fi
fi

install_skill_to() {
  local target_dir="$1"
  local label="$2"
  mkdir -p "${target_dir}"
  rm -rf "${target_dir}/ntts-flightlog"
  cp -R "${REPO_ROOT}/skill/ntts-flightlog" "${target_dir}/ntts-flightlog"
  chmod +x "${target_dir}/ntts-flightlog/scripts/flightlog.sh"
  printf 'Installed %s skill: %s/ntts-flightlog\n' "${label}" "${target_dir}"
}

(( INSTALL_CODEX  )) && install_skill_to "${CODEX_SKILLS_DIR}"  "Codex"
(( INSTALL_CLAUDE )) && install_skill_to "${CLAUDE_SKILLS_DIR}" "Claude Code"
(( INSTALL_GEMINI )) && install_skill_to "${GEMINI_SKILLS_DIR}" "Gemini CLI"

if (( INSTALL_CLI )); then
  mkdir -p "${LOCAL_BIN_DIR}"
  cp "${REPO_ROOT}/bin/ntts-flightlog" "${LOCAL_BIN_DIR}/ntts-flightlog"
  chmod +x "${LOCAL_BIN_DIR}/ntts-flightlog"
  printf 'Installed CLI: %s\n' "${LOCAL_BIN_DIR}/ntts-flightlog"
  case ":${PATH}:" in
    *":${LOCAL_BIN_DIR}:"*) : ;;
    *) printf 'Note: %s is not on $PATH. Add it to your shell rc to call "ntts-flightlog" directly.\n' "${LOCAL_BIN_DIR}" ;;
  esac
fi

printf 'Restart your agent (Codex/Claude Code/Gemini) to reload the skill list.\n'

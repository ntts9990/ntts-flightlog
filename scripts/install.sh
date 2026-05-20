#!/usr/bin/env bash
# flightlog installer: downloads Go binary from GitHub Releases + installs SKILL.md
# Usage: install.sh [--codex|--claude|--gemini|--all] [--no-cli] [-h]
# Env:   LOCAL_BIN_DIR  CODEX/CLAUDE/GEMINI_SKILLS_DIR  INSTALL_PREFIX (dry-run)
set -euo pipefail
REPO="ntts9990/ntts-flightlog"
P="${INSTALL_PREFIX:-$HOME}"; LOCAL_BIN_DIR="${LOCAL_BIN_DIR:-$P/.local/bin}"
CODEX_SKILLS_DIR="${CODEX_SKILLS_DIR:-$P/.codex/skills}"
CLAUDE_SKILLS_DIR="${CLAUDE_SKILLS_DIR:-$P/.claude/skills}"
GEMINI_SKILLS_DIR="${GEMINI_SKILLS_DIR:-$P/.gemini/skills}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
die() { printf 'error: %s\n' "$1" >&2; exit 1; }
CLI=1; CODEX=0; CLAUDE=0; GEMINI=0; EXP=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --codex)  CODEX=1; EXP=1 ;;   --claude) CLAUDE=1; EXP=1 ;;
    --gemini) GEMINI=1; EXP=1 ;;  --all)    CODEX=1; CLAUDE=1; GEMINI=1; EXP=1 ;;
    --no-cli) CLI=0 ;;
    -h|--help) sed -n '2,4p' "$0" | sed 's/^# \?//'; exit 0 ;;
    *) die "Unknown option: $1" ;;
  esac; shift
done
if (( !EXP )); then
  [[ -d "$P/.codex"  ]] && CODEX=1  || true
  [[ -d "$P/.claude" ]] && CLAUDE=1 || true
  [[ -d "$P/.gemini" ]] && GEMINI=1 || true
  (( CODEX || CLAUDE || GEMINI )) || CODEX=1
fi
install_cli() {
  local os arch; os="$(uname -s | tr '[:upper:]' '[:lower:]')"; arch="$(uname -m)"
  case "$arch" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) die "Unsupported arch: $arch" ;; esac
  case "$os"  in darwin|linux) ;; msys*|cygwin*|mingw*) os=windows ;; *) die "Unsupported OS: $os" ;; esac
  [[ "$os" == windows && "$arch" == arm64 ]] && die "windows/arm64 not supported"
  printf 'Fetching latest flightlog release...\n'
  local tag; tag="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null \
    | grep '"tag_name"' | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')" || true
  [[ -z "$tag" ]] && { printf 'Warning: no release found for %s — skipping binary install.\n' "$REPO"; return 0; }
  local ver ext base url tmp
  ver="${tag#v}"; ext=tar.gz; [[ "$os" == windows ]] && ext=zip
  base="flightlog_${ver}_${os}_${arch}"; url="https://github.com/$REPO/releases/download/$tag"
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' RETURN
  printf 'Downloading %s.%s...\n' "$base" "$ext"
  curl -fsSL "$url/$base.$ext" -o "$tmp/$base.$ext"
  curl -fsSL "$url/checksums.txt" -o "$tmp/checksums.txt"
  local want got
  want="$(grep "[[:space:]]${base}[.]${ext}$" "$tmp/checksums.txt" | awk '{print $1}')"
  [[ -z "$want" ]] && die "Checksum not found for $base.$ext"
  if command -v sha256sum >/dev/null 2>&1; then got="$(sha256sum "$tmp/$base.$ext" | awk '{print $1}')"
  else got="$(shasum -a 256 "$tmp/$base.$ext" | awk '{print $1}')"; fi
  [[ "$got" == "$want" ]] || die "Checksum mismatch (got $got, want $want)"
  mkdir -p "$LOCAL_BIN_DIR"
  if [[ "$ext" == zip ]]; then
    unzip -q "$tmp/$base.$ext" flightlog.exe -d "$tmp"; cp "$tmp/flightlog.exe" "$LOCAL_BIN_DIR/"
    printf 'Installed flightlog %s → %s/flightlog.exe\n' "$tag" "$LOCAL_BIN_DIR"
  else
    tar -xzf "$tmp/$base.$ext" -C "$tmp" flightlog; cp "$tmp/flightlog" "$LOCAL_BIN_DIR/"; chmod +x "$LOCAL_BIN_DIR/flightlog"
    printf 'Installed flightlog %s → %s/flightlog\n' "$tag" "$LOCAL_BIN_DIR"
  fi
  case ":$PATH:" in *":$LOCAL_BIN_DIR:"*) ;; *) printf 'Note: add %s to $PATH\n' "$LOCAL_BIN_DIR" ;; esac
}
install_skill() {
  mkdir -p "$1"; rm -rf "$1/ntts-flightlog"; cp -R "$REPO_ROOT/skill/ntts-flightlog" "$1/ntts-flightlog"
  printf 'Installed %s skill: %s/ntts-flightlog\n' "$2" "$1"
}
(( CLI    )) && install_cli
(( CODEX  )) && install_skill "$CODEX_SKILLS_DIR"  "Codex"
(( CLAUDE )) && install_skill "$CLAUDE_SKILLS_DIR" "Claude Code"
(( GEMINI )) && install_skill "$GEMINI_SKILLS_DIR" "Gemini CLI"
printf 'Done. Restart your agent to reload the skill list.\n'

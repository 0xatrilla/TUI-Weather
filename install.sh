#!/usr/bin/env bash
# TUI Weather installer.
#
#   curl -fsSL https://raw.githubusercontent.com/0xatrilla/TUI-Weather/main/install.sh | bash
#
# Downloads the latest prebuilt `tuiwthr` binary for your platform. If no
# prebuilt binary is available it falls back to building from source (needs Go).
set -euo pipefail

OWNER="0xatrilla"
REPO="TUI-Weather"
BIN="tuiwthr"

info() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; }

# --- pick an install dir on PATH (no sudo when possible) ---
choose_bindir() {
  for d in "$HOME/.local/bin" "$HOME/bin"; do
    case ":$PATH:" in *":$d:"*) echo "$d"; return;; esac
  done
  if [ -w /usr/local/bin ]; then echo /usr/local/bin; return; fi
  echo "$HOME/.local/bin"
}
BINDIR="$(choose_bindir)"
mkdir -p "$BINDIR"

# --- detect platform ---
os="$(uname -s)"; arch="$(uname -m)"
case "$os" in
  Linux)  os=linux ;;
  Darwin) os=darwin ;;
  *)      os="" ;;
esac
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *)             arch="" ;;
esac

build_from_source() {
  info "Falling back to build-from-source"
  command -v go  >/dev/null 2>&1 || { err "Go is required to build from source: https://go.dev/dl/"; exit 1; }
  command -v git >/dev/null 2>&1 || { err "git is required to build from source."; exit 1; }
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
  info "Cloning https://github.com/$OWNER/$REPO"
  git clone --depth 1 "https://github.com/$OWNER/$REPO.git" "$tmp/src" >/dev/null 2>&1
  info "Building $BIN…"
  ( cd "$tmp/src" && go build -o "$BIN" . )
  install -m 0755 "$tmp/src/$BIN" "$BINDIR/$BIN"
}

install_prebuilt() {
  [ -n "$os" ] && [ -n "$arch" ] || return 1
  command -v curl >/dev/null 2>&1 || return 1
  asset="${BIN}_${os}_${arch}.tar.gz"
  url="https://github.com/$OWNER/$REPO/releases/latest/download/$asset"
  tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
  info "Downloading $asset"
  curl -fsSL "$url" -o "$tmp/$asset" || return 1
  tar -xzf "$tmp/$asset" -C "$tmp" || return 1
  [ -f "$tmp/$BIN" ] || return 1
  install -m 0755 "$tmp/$BIN" "$BINDIR/$BIN"
}

if install_prebuilt; then
  info "Installed prebuilt $BINDIR/$BIN"
else
  build_from_source
  info "Installed $BINDIR/$BIN"
fi

# --- PATH hint ---
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) err "$BINDIR is not on your PATH. Add this to your shell profile:"
     printf '\n    export PATH="%s:$PATH"\n\n' "$BINDIR" ;;
esac

info "Done! Run:  $BIN"

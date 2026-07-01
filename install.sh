#!/usr/bin/env bash
# TUI Weather installer — builds from source and installs the `tuiwthr` command.
#
#   curl -fsSL https://raw.githubusercontent.com/0xatrilla/TUI-Weather/main/install.sh | bash
#
set -euo pipefail

REPO="https://github.com/0xatrilla/TUI-Weather.git"
BIN="tuiwthr"

info() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
err()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; }

# --- prerequisites ---
if ! command -v go >/dev/null 2>&1; then
  err "Go is required but not found. Install it from https://go.dev/dl/ and re-run."
  exit 1
fi
if ! command -v git >/dev/null 2>&1; then
  err "git is required but not found."
  exit 1
fi

# --- pick an install dir on PATH (no sudo if possible) ---
choose_bindir() {
  for d in "$HOME/.local/bin" "$HOME/bin"; do
    case ":$PATH:" in *":$d:"*) echo "$d"; return;; esac
  done
  if [ -w /usr/local/bin ]; then echo /usr/local/bin; return; fi
  echo "$HOME/.local/bin" # fall back; we'll warn about PATH
}
BINDIR="$(choose_bindir)"
mkdir -p "$BINDIR"

# --- build in a temp clone ---
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
info "Cloning $REPO"
git clone --depth 1 "$REPO" "$TMP/src" >/dev/null 2>&1

info "Building $BIN (this may take a moment)…"
( cd "$TMP/src" && go build -o "$BIN" . )

install -m 0755 "$TMP/src/$BIN" "$BINDIR/$BIN"
info "Installed $BINDIR/$BIN"

# --- PATH hint ---
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) err "$BINDIR is not on your PATH. Add this to your shell profile:"
     printf '\n    export PATH="%s:$PATH"\n\n' "$BINDIR" ;;
esac

info "Done! Run:  $BIN"

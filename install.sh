#!/bin/sh
# Pushwoosh SDK Integrator: downloads the integrator binary of this release, checks its sha256
# against checksums.txt from the same release, then runs it with the install code.
# Usage: curl -fsSL https://github.com/Pushwoosh/SDK-Integrator/releases/latest/download/install.sh | sh -s -- <code> [claude|cursor]
# Everything runs from main on the last line: a truncated download runs nothing.
set -eu

REPO='Pushwoosh/SDK-Integrator'
# Pinned by the release workflow to the tag this script ships in; "latest" only when run from a checkout.
VERSION='latest'

say() { printf '%s\n' "$*"; }
die() {
  printf 'pushwoosh install: %s\n' "$*" >&2
  exit 1
}
cleanup() { [ -z "${TMP_DIR:-}" ] || rm -rf "$TMP_DIR"; }

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    die "neither sha256sum nor shasum found; cannot verify the download"
  fi
}

main() {
  [ "$#" -ge 1 ] && [ -n "$1" ] || die "missing install code; copy the full command from the Control Panel"

  case $(uname -s) in
    Linux) OS=linux ;;
    Darwin) OS=darwin ;;
    *) die "unsupported OS $(uname -s); Windows is not supported yet" ;;
  esac
  case $(uname -m) in
    x86_64 | amd64) ARCH=amd64 ;;
    arm64 | aarch64) ARCH=arm64 ;;
    *) die "unsupported CPU $(uname -m)" ;;
  esac
  command -v curl >/dev/null 2>&1 || die "curl not found on PATH"

  if [ "$VERSION" = latest ]; then
    BASE="https://github.com/${REPO}/releases/latest/download"
  else
    BASE="https://github.com/${REPO}/releases/download/${VERSION}"
  fi
  NAME="pushwoosh-integrator_${OS}_${ARCH}"

  TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t pushwoosh) || die "cannot create a temp directory"
  trap cleanup EXIT
  trap 'exit 130' INT TERM

  say "==> downloading ${NAME} (${VERSION})"
  curl -fsSL "${BASE}/${NAME}" -o "${TMP_DIR}/${NAME}" </dev/null || die "cannot download ${BASE}/${NAME}"
  curl -fsSL "${BASE}/checksums.txt" -o "${TMP_DIR}/checksums.txt" </dev/null || die "cannot download checksums.txt"

  WANT=$(awk -v n="$NAME" '$2 == n { print $1 }' "${TMP_DIR}/checksums.txt")
  [ -n "$WANT" ] || die "checksums.txt has no entry for ${NAME}"
  [ "$(sha256 "${TMP_DIR}/${NAME}")" = "$WANT" ] || die "the downloaded binary does not match its checksum"
  chmod +x "${TMP_DIR}/${NAME}"

  # stdin is this script; the integrator asks its questions on /dev/tty.
  "${TMP_DIR}/${NAME}" "$@" </dev/null
}

main "$@"

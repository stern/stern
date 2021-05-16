#!/usr/bin/env bash

set -eo pipefail; [[ -n "$DEBUG" ]] && set -ux

ROOT_DIR="$(cd "$(dirname $0)" && pwd)/.."
cd "$ROOT_DIR"

tempfile="$(mktemp)"
cat README.md >"$tempfile"
make update-readme README_FILE="$tempfile" >/dev/null
diff="$(diff -u ./README.md "$tempfile" ||:)"

if [[ -n "$diff" ]]; then
  echo "$diff" >&2
  echo "Error: Running update-readme made a difference in README.md." >&2
  echo "Maybe you forgot to run 'make update-readme'." >&2
  exit 1
fi

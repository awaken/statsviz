#!/bin/bash
set -euo pipefail

# Check that the files in dist.zip are up to date with the files in dist/
# This must be run after npm run build.
cd "$(dirname "$0")/.."

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT
unzip -qq dist.zip -d "$tmp_dir"

if diff -qr "$tmp_dir/dist" dist >/dev/null; then
  echo "'dist.zip' contains up to date files."
else
  echo "'dist.zip' does not contain up to date files, did you forget to build/commit the assets?" && exit 1
fi

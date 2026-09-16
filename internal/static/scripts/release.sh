#!/bin/sh
set -eu
cd "$(dirname "$0")/.."

# Releases run offline in the complete, immutable image selected by make release.
case "${STATSVIZ_RELEASE_IMAGE:-}" in
  sha256:*|*@sha256:*) ;;
  *) echo 'Use make release RELEASE_IMAGE=<immutable image digest>' >&2; exit 2 ;;
esac
npm ci --offline --no-audit --no-fund
rm -rf dist
npm run build
sh scripts/zip.sh

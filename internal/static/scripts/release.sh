#!/bin/bash
set -eu

# Zip the dist directory into dist.zip
cd "$(dirname "$0")/.."

rm -f dist.zip
rm -rf ./dist
npm run build
zip -r dist.zip dist/*

#!/bin/sh
set -eu
src=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
root=$(CDPATH= cd -- "$src/../../../../.." && pwd)
dir=$(mktemp -d "$root/tmp/statsviz-make-XXXXXX")
trap 'rm -rf "$dir"' EXIT HUP INT TERM
cp "$src/Makefile" "$src/Dockerfile" "$src/docker-entrypoint.sh" "$dir/"
mkdir "$dir/bin"
cat > "$dir/bin/docker" <<'MOCK'
#!/bin/sh
printf '%s\n' "$*" >> "$BUILD_LOG"
MOCK
chmod 700 "$dir/bin/docker"
export PATH="$dir/bin:$PATH" BUILD_LOG="$dir/build.log"
# A stale stamp must not hide a deleted image or a changed build argument.
touch "$dir/.docker-image-stamp"
make -s -C "$dir" image IMAGE_TAG=fixture-first NODE_IMAGE=fixture-base-one
make -s -C "$dir" image IMAGE_TAG=fixture-second NODE_IMAGE=fixture-base-two
test -f "$BUILD_LOG" || { echo "stale stamp suppressed docker build" >&2; exit 1; }
test "$(wc -l < "$BUILD_LOG" | tr -d ' ')" = 2
grep -q 'NODE_IMAGE=fixture-base-two -t fixture-second' "$BUILD_LOG"
printf '%s\n' 'Docker cache regression passed'
# A release must never silently use the development image or a floating tag.
if make -s -C "$dir" release RELEASE_IMAGE=fixture:latest 2>/dev/null; then
  echo 'floating release image accepted' >&2; exit 1
fi
test "$(wc -l < "$BUILD_LOG" | tr -d ' ')" = 2
make -s -C "$dir" release RELEASE_IMAGE=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
grep -q -- '--pull=never --network=none' "$BUILD_LOG"
grep -q 'sh scripts/release.sh' "$BUILD_LOG"

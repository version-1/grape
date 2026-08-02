#!/usr/bin/env sh

set -eu

version="${VERSION:-}"
commit="${COMMIT:-$(git rev-parse --short HEAD)}"
release_dir="${RELEASE_DIR:-dist}"

if [ -z "$version" ]; then
  echo "VERSION is required (for example: make release VERSION=0.1.2)" >&2
  exit 1
fi

if ! printf '%s\n' "$version" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  echo "VERSION must be an unprefixed MAJOR.MINOR.PATCH version: $version" >&2
  exit 1
fi

for asset in grape_darwin_arm64 grape_linux_amd64; do
  if [ -e "$release_dir/$asset" ] || [ -L "$release_dir/$asset" ]; then
    echo "Refusing to overwrite $release_dir/$asset" >&2
    exit 1
  fi
done

staging_dir="$(mktemp -d)"
trap 'rm -rf "$staging_dir"' EXIT HUP INT TERM

build_asset() {
  goos="$1"
  goarch="$2"
  output="$3"

  echo "Building $output"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build \
      -ldflags "-X main.version=$version -X main.commit=$commit" \
      -o "$staging_dir/$output" \
      ./cmd/grape
}

build_asset darwin arm64 grape_darwin_arm64
build_asset linux amd64 grape_linux_amd64

mkdir -p "$release_dir"
mv "$staging_dir/grape_darwin_arm64" "$release_dir/grape_darwin_arm64"
mv "$staging_dir/grape_linux_amd64" "$release_dir/grape_linux_amd64"

echo "Release binaries created in $release_dir"

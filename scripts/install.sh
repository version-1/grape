#!/usr/bin/env sh

set -eu

repository="version-1/grape"

case "${GRAPE_VERSION:-}" in
  v*)
    echo "GRAPE_VERSION must not start with v: $GRAPE_VERSION" >&2
    exit 1
    ;;
esac

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) asset="grape_darwin_arm64" ;;
  Linux-x86_64) asset="grape_linux_amd64" ;;
  *)
    echo "Unsupported platform: $(uname -s)-$(uname -m)" >&2
    exit 1
    ;;
esac

if [ -e "./grape" ] || [ -L "./grape" ]; then
  echo "Refusing to overwrite ./grape" >&2
  exit 1
fi

if [ -n "${GRAPE_VERSION:-}" ]; then
  release_path="download/$GRAPE_VERSION"
  release_label="$GRAPE_VERSION"
else
  release_path="latest/download"
  release_label="latest release"
fi

download_url="https://github.com/$repository/releases/$release_path/$asset"
temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM
curl --fail --location --silent --show-error "$download_url" --output "$temporary_dir/grape"
mv "$temporary_dir/grape" ./grape

echo "Downloaded grape $release_label to ./grape"

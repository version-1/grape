#!/usr/bin/env sh

set -eu

version="${GW_VERSION:-0.1.0}"
repository="version-1/grape"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) asset="grape_${version}_darwin_arm64" ;;
  Linux-x86_64) asset="grape_${version}_linux_amd64" ;;
  *)
    echo "Unsupported platform: $(uname -s)-$(uname -m)" >&2
    exit 1
    ;;
esac

if [ -e "./grape" ] || [ -L "./grape" ]; then
  echo "Refusing to overwrite ./grape" >&2
  exit 1
fi

download_url="https://github.com/$repository/releases/download/$version/$asset"
temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM
curl --fail --location --silent --show-error "$download_url" --output "$temporary_dir/grape"
mv "$temporary_dir/grape" ./grape

echo "Downloaded grape $version to ./grape"

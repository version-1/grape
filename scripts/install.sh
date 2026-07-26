#!/usr/bin/env sh

set -eu

version="${GW_VERSION:-0.1.0}"
repository="version-1/grape"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) asset="gw_${version}_darwin_arm64" ;;
  Linux-x86_64) asset="gw_${version}_linux_amd64" ;;
  *)
    echo "Unsupported platform: $(uname -s)-$(uname -m)" >&2
    exit 1
    ;;
esac

if [ -e "./grape" ]; then
	echo "Refusing to overwrite ./grape" >&2
  exit 1
fi

download_url="https://github.com/$repository/releases/download/$version/$asset"
curl --fail --location --silent --show-error "$download_url" --output ./grape

echo "Downloaded grape $version to ./grape"

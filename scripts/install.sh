#!/usr/bin/env sh

set -eu

version="${GW_VERSION:-0.1.0}"
repository="version-1/grape"
install_dir="${GW_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) asset="gw_${version}_darwin_arm64" ;;
  Linux-x86_64) asset="gw_${version}_linux_amd64" ;;
  *)
    echo "Unsupported platform: $(uname -s)-$(uname -m)" >&2
    exit 1
    ;;
esac

if [ -e "$install_dir/gw" ]; then
  echo "Refusing to overwrite $install_dir/gw" >&2
  exit 1
fi

mkdir -p "$install_dir"
temporary_dir="$(mktemp -d)"
trap 'rm -rf "$temporary_dir"' EXIT HUP INT TERM

download_url="https://github.com/$repository/releases/download/$version/$asset"
curl --fail --location --silent --show-error "$download_url" --output "$temporary_dir/gw"
chmod 0755 "$temporary_dir/gw"
mv "$temporary_dir/gw" "$install_dir/gw"

echo "Installed gw $version to $install_dir/gw"

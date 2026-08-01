#!/usr/bin/env sh

set -eu

repository_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

fake_bin="$test_root/bin"
mkdir -p "$fake_bin"

cat >"$fake_bin/uname" <<'EOF'
#!/usr/bin/env sh
case "$1" in
  -s) printf '%s\n' "${TEST_UNAME_SYSTEM:-Linux}" ;;
  -m) printf '%s\n' "${TEST_UNAME_MACHINE:-x86_64}" ;;
  *) exit 2 ;;
esac
EOF

cat >"$fake_bin/curl" <<'EOF'
#!/usr/bin/env sh
output=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output="$2"
      shift 2
      ;;
    http://*|https://*)
      url="$1"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

printf '%s\n' "$url" >"$TEST_REQUEST_LOG"
printf '%s\n' 'binary' >"$output"
EOF

chmod +x "$fake_bin/uname" "$fake_bin/curl"

run_install_test() {
  name="$1"
  expected_url="$2"
  version="${3:-}"
  test_dir="$test_root/$name"
  request_log="$test_dir/request.log"
  mkdir -p "$test_dir"

  if [ -n "$version" ]; then
    (
      cd "$test_dir"
      PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$request_log" GRAPE_VERSION="$version" \
        sh "$repository_root/scripts/install.sh"
    )
  else
    (
      cd "$test_dir"
      PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$request_log" \
        sh "$repository_root/scripts/install.sh"
    )
  fi

  actual_url="$(cat "$request_log")"
  if [ "$actual_url" != "$expected_url" ]; then
    echo "$name: download URL = $actual_url, want $expected_url" >&2
    exit 1
  fi
  if [ ! -f "$test_dir/grape" ]; then
    echo "$name: grape was not installed" >&2
    exit 1
  fi
}

run_install_test \
  latest-release \
  "https://github.com/version-1/grape/releases/latest/download/grape_linux_amd64"

run_install_test \
  explicit-release \
  "https://github.com/version-1/grape/releases/download/0.1.2/grape_linux_amd64" \
  "0.1.2"

echo "install tests passed"

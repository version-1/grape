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
has_fail=false
has_location=false
has_silent=false
has_show_error=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output="$2"
      shift 2
      ;;
    --fail) has_fail=true; shift ;;
    --location) has_location=true; shift ;;
    --silent) has_silent=true; shift ;;
    --show-error) has_show_error=true; shift ;;
    http://*|https://*)
      url="$1"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

if [ "$has_fail" != true ] || [ "$has_location" != true ] || \
  [ "$has_silent" != true ] || [ "$has_show_error" != true ]; then
  echo "required curl safety option missing" >&2
  exit 2
fi
if [ "${TEST_CURL_FAIL:-}" = "1" ]; then
  exit 22
fi

printf '%s\n' "$url" >"$TEST_REQUEST_LOG"
printf '%s\n' 'binary' >"$output"
EOF

chmod +x "$fake_bin/uname" "$fake_bin/curl"

run_install_test() {
  name="$1"
  expected_url="$2"
  version="${3:-}"
  system="${4:-Linux}"
  machine="${5:-x86_64}"
  test_dir="$test_root/$name"
  request_log="$test_dir/request.log"
  mkdir -p "$test_dir"

  if [ -n "$version" ]; then
    (
      cd "$test_dir"
      PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$request_log" GRAPE_VERSION="$version" \
        TEST_UNAME_SYSTEM="$system" TEST_UNAME_MACHINE="$machine" \
        sh "$repository_root/scripts/install.sh"
    )
  else
    (
      cd "$test_dir"
      PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$request_log" \
        TEST_UNAME_SYSTEM="$system" TEST_UNAME_MACHINE="$machine" \
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

run_install_test \
  darwin-latest \
  "https://github.com/version-1/grape/releases/latest/download/grape_darwin_arm64" \
  "" \
  "Darwin" \
  "arm64"

unsupported_dir="$test_root/unsupported-platform"
mkdir -p "$unsupported_dir"
if (
  cd "$unsupported_dir"
  PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$unsupported_dir/request.log" \
    TEST_UNAME_SYSTEM="FreeBSD" TEST_UNAME_MACHINE="amd64" \
    sh "$repository_root/scripts/install.sh"
); then
  echo "unsupported platform was accepted" >&2
  exit 1
fi

overwrite_dir="$test_root/overwrite"
mkdir -p "$overwrite_dir"
printf '%s\n' 'existing' >"$overwrite_dir/grape"
if (
  cd "$overwrite_dir"
  PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$overwrite_dir/request.log" \
    sh "$repository_root/scripts/install.sh"
); then
  echo "existing grape was overwritten" >&2
  exit 1
fi
if [ "$(cat "$overwrite_dir/grape")" != "existing" ]; then
  echo "existing grape contents changed" >&2
  exit 1
fi

failure_dir="$test_root/download-failure"
mkdir -p "$failure_dir"
if (
  cd "$failure_dir"
  PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$failure_dir/request.log" TEST_CURL_FAIL=1 \
    sh "$repository_root/scripts/install.sh"
); then
  echo "curl failure was ignored" >&2
  exit 1
fi
if [ -e "$failure_dir/grape" ] || [ -L "$failure_dir/grape" ]; then
  echo "curl failure left a grape output" >&2
  exit 1
fi

prefixed_dir="$test_root/prefixed-version"
mkdir -p "$prefixed_dir"
if (
  cd "$prefixed_dir"
  PATH="$fake_bin:$PATH" TEST_REQUEST_LOG="$prefixed_dir/request.log" GRAPE_VERSION=v0.1.2 \
    sh "$repository_root/scripts/install.sh"
); then
  echo "v-prefixed GRAPE_VERSION was accepted" >&2
  exit 1
fi

echo "install tests passed"

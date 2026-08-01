# Release Guide

This guide is the canonical procedure for building and publishing grape releases.

## Release Contract

- Release tags and titles use an unprefixed semantic version such as `0.1.2`, never `v0.1.2`.
- A release targets the exact latest commit on `origin/main`.
- Every published release contains both stable asset names:
  - `grape_darwin_arm64`
  - `grape_linux_amd64`
- The installer depends on these stable names. It uses GitHub's latest-release URL by default and the exact tag when `GRAPE_VERSION` is set.
- Create the GitHub release as a draft, upload and verify both assets, and only then publish it. Publishing an asset-less release temporarily breaks the default installer.

## 1. Prepare the release commit

Start from a clean working tree and update the remote references:

```sh
git status --short --branch
git fetch origin
git rev-parse origin/main
```

Use the resulting full commit hash as the release target. Do not release an unmerged branch or a stale local `main`.

Confirm that neither the tag nor the GitHub release already exists. For version `0.1.2`:

```sh
git ls-remote --tags origin refs/tags/0.1.2
gh release view 0.1.2
```

If either exists, inspect it instead of overwriting, moving, or deleting it. Use a new patch version unless correcting the existing release was explicitly approved.

## 2. Build the assets

Build both supported platforms with the unprefixed version:

```sh
make release VERSION=0.1.2
```

The command cross-compiles static binaries with `CGO_ENABLED=0`, embeds the version and current short commit, and writes:

```text
dist/grape_darwin_arm64
dist/grape_linux_amd64
```

The build stages both binaries in a temporary directory and refuses to overwrite either existing file in `dist`. This prevents a failed or repeated build from leaving an unnoticed mixture of release assets. Remove or archive a previous `dist` directory deliberately before rebuilding.

Inspect the outputs and record checksums:

```sh
file dist/grape_darwin_arm64 dist/grape_linux_amd64
shasum -a 256 dist/grape_darwin_arm64 dist/grape_linux_amd64
```

On macOS arm64, also verify the embedded build information:

```sh
./dist/grape_darwin_arm64 version
```

## 3. Create a draft release

Create the release against the exact full `origin/main` commit recorded earlier:

```sh
gh release create 0.1.2 \
  --target <full-origin-main-commit> \
  --title 0.1.2 \
  --generate-notes \
  --draft
```

Upload the stable assets without `--clobber`:

```sh
gh release upload 0.1.2 \
  dist/grape_darwin_arm64 \
  dist/grape_linux_amd64
```

Leaving out `--clobber` prevents an existing asset from being silently replaced.

## 4. Verify before publishing

Confirm the target, draft state, and exact asset names:

```sh
gh release view 0.1.2 \
  --json tagName,targetCommitish,isDraft,isPrerelease,assets,url
```

Download the draft assets to a new temporary directory and compare their checksums with the local build. Do not publish if the target or either checksum differs.

After verification, publish the release:

```sh
gh release edit 0.1.2 --draft=false
```

Confirm that it is published, is not a prerelease, and still has both assets:

```sh
gh release view 0.1.2 \
  --json tagName,targetCommitish,isDraft,isPrerelease,publishedAt,assets,url
```

## 5. Test the installer

Run the installer from an empty temporary directory. Test both latest resolution and the explicit version without overwriting an existing binary:

```sh
install_test_dir="$(mktemp -d)"
cd "$install_test_dir"

curl --fail --location --silent --show-error \
  https://raw.githubusercontent.com/version-1/grape/main/scripts/install.sh \
  --output install.sh

sh install.sh
rm ./grape
GRAPE_VERSION=0.1.2 sh install.sh
./grape version
```

The reported version must be `0.1.2`, and both downloads must use the binaries verified before publication.

## Recovery

If validation fails while the release is still a draft, leave it unpublished while investigating. Do not publish incomplete assets.

Do not move a published tag, replace a published asset, or delete a published release without explicit approval. Prefer fixing the release process and publishing a new patch version.

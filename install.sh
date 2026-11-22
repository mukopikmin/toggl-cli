#!/bin/sh

set -eu

repository="https://github.com/mukopikmin/toggl-cli"
install_dir="${HOME:-}/.local/bin"
temporary_dir=""
staged_binary=""

cleanup() {
  if [ -n "$staged_binary" ]; then
    rm -f "$staged_binary"
  fi
  if [ -n "$temporary_dir" ]; then
    rm -rf "$temporary_dir"
  fi
}

fail() {
  echo "toggl installer: $*" >&2
  exit 1
}

usage() {
  echo "Usage: install.sh [--nightly]" >&2
}

trap cleanup EXIT HUP INT TERM

if [ "$#" -gt 1 ]; then
  usage
  exit 1
fi

channel="latest"
if [ "$#" -eq 1 ]; then
  if [ "$1" != "--nightly" ]; then
    usage
    exit 1
  fi
  channel="nightly"
fi

[ -n "${HOME:-}" ] || fail "HOME is not set"

for command in curl tar mktemp; do
  command -v "$command" >/dev/null 2>&1 || fail "required command not found: $command"
done

case "$(uname -s):$(uname -m)" in
  Darwin:arm64 | Darwin:aarch64)
    target="darwin-arm64"
    verifier_asset="cosign-darwin-arm64"
    verifier_sha="5fadd012ae6381a6a29ff86a7d39aa873878852f1073fc90b15995961ecfb084"
    ;;
  Linux:x86_64 | Linux:amd64)
    target="linux-x64"
    verifier_asset="cosign-linux-amd64"
    verifier_sha="c956e5dfcac53d52bcf058360d579472f0c1d2d9b69f55209e256fe7783f4c74"
    ;;
  *)
    fail "unsupported platform: $(uname -s) $(uname -m)"
    ;;
esac

if [ "$channel" = "nightly" ]; then
  tag="nightly"
else
  release_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "$repository/releases/latest")" ||
    fail "could not resolve the latest release"
  tag="${release_url##*/}"
fi

case "$tag" in
  nightly) ;;
  v[0-9]*.[0-9]*.[0-9]* | [0-9]*.[0-9]*.[0-9]*) ;;
  *) fail "invalid release tag: $tag" ;;
esac

version="${tag#v}"
if [ "$version" = "nightly" ]; then
  archive_root="toggl-cli-nightly-${target}"
else
  archive_root="toggl-cli-v${version}-${target}"
fi
archive="${archive_root}.tar.gz"
download_url="$repository/releases/download/$tag"
temporary_dir="$(mktemp -d "${TMPDIR:-/tmp}/toggl-install.XXXXXX")"
archive_path="$temporary_dir/$archive"
checksum_path="$archive_path.sha256"

echo "Downloading toggl $version for $target..."
curl -fsSL -o "$archive_path" "$download_url/$archive" ||
  fail "could not download $archive"
curl -fsSL -o "$checksum_path" "$download_url/$archive.sha256" ||
  fail "could not download $archive.sha256"

expected_checksum="$(tr -d '[:space:]' < "$checksum_path")"
case "$expected_checksum" in
  *[!0-9a-fA-F]* | "") fail "invalid SHA-256 checksum" ;;
esac
[ "${#expected_checksum}" -eq 64 ] || fail "invalid SHA-256 checksum"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}
actual_checksum="$(sha256_file "$archive_path")"
[ "$actual_checksum" = "$expected_checksum" ] || fail "SHA-256 checksum mismatch"

repo=mukopikmin/toggl-cli
workflow=release
source_ref="refs/tags/$tag"
if [ "$tag" = nightly ]; then
  workflow=nightly
  source_ref=refs/heads/main
fi
commit="$(curl -fsSL -H 'Accept: application/vnd.github.sha' "https://api.github.com/repos/$repo/commits/$tag")" || fail "could not resolve release commit"
case "$commit" in *[!0-9a-f]*) fail "invalid release commit";; esac
[ "${#commit}" -eq 40 ] || fail "invalid release commit"
bundle="$temporary_dir/provenance.json"
curl -fsSL -o "$bundle" "$download_url/provenance.json" || fail "could not download build provenance"

if command -v cosign >/dev/null 2>&1; then
  verifier=cosign
else
  verifier="$temporary_dir/cosign"
  curl -fsSL -o "$verifier" "https://github.com/sigstore/cosign/releases/download/v3.0.6/$verifier_asset" || fail "could not download verifier"
  [ "$(sha256_file "$verifier")" = "$verifier_sha" ] || fail "verifier checksum mismatch"
  chmod 700 "$verifier"
fi
"$verifier" verify-blob-attestation --bundle "$bundle" --type slsaprovenance1 \
  --certificate-identity "$repository/.github/workflows/$workflow.yml@$source_ref" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-github-workflow-sha "$commit" "$archive_path" || fail "release provenance verification failed"

tar -xzf "$archive_path" -C "$temporary_dir" || fail "could not extract $archive"
binary_path="$temporary_dir/${archive_root}/toggl"
[ -f "$binary_path" ] || fail "release archive does not contain toggl"

mkdir -p "$install_dir"
staged_binary="$(mktemp "$install_dir/.toggl.XXXXXX")"
cp "$binary_path" "$staged_binary"
chmod 755 "$staged_binary"
mv "$staged_binary" "$install_dir/toggl"
staged_binary=""

echo "Installed toggl $version to $install_dir/toggl"

# Older releases remain installable without the completion installer.
completion_installer="$temporary_dir/$archive_root/install-completions.sh"
if [ -f "$completion_installer" ]; then
  sh "$completion_installer" || echo "Completion installation failed; the Toggl binary was installed successfully." >&2
fi

case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to your PATH to run toggl." ;;
esac

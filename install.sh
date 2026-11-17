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
    ;;
  Linux:x86_64 | Linux:amd64)
    target="linux-x64"
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

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "$archive_path" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
else
  fail "required command not found: sha256sum or shasum"
fi

[ "$actual_checksum" = "$expected_checksum" ] || fail "SHA-256 checksum mismatch"

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

# Completion installation is best-effort: a missing completion framework or an
# unwritable directory must not undo a successful binary installation. Never
# edit shell startup files or request elevated privileges.
install_completion() {
  source_path="$temporary_dir/$archive_root/completions/$1"
  destination="$2"
  [ -f "$source_path" ] || return 0 # Older releases have no completion files.
  if (
    mkdir -p "$(dirname "$destination")" || exit 1
    completion_stage="$(mktemp "$(dirname "$destination")/.toggl-completion.XXXXXX")" || exit 1
    trap 'rm -f "$completion_stage"' EXIT HUP INT TERM
    cp "$source_path" "$completion_stage" &&
    chmod 644 "$completion_stage" &&
    mv "$completion_stage" "$destination"
  ); then
    echo "Installed $1 completion to $destination"
  else
    echo "Could not install $1 completion to $destination; retry with 'toggl completion install $1'." >&2
  fi
}

data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
install_completion bash "${BASH_COMPLETION_USER_DIR:-$data_dir/bash-completion}/completions/toggl"
install_completion fish "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions/toggl.fish"

if [ -f "$temporary_dir/$archive_root/completions/zsh" ]; then
  zsh_completion_dir=""
  if command -v zsh >/dev/null 2>&1; then
    # -f avoids loading user startup files. Choose an existing writable site
    # directory already in zsh's default fpath (including Homebrew builds).
    zsh_paths="$(zsh -fc 'printf "%s\n" "${fpath[@]}"' 2>/dev/null)" || zsh_paths=""
    while IFS= read -r directory; do
      case "$directory" in
        */site-functions)
          if [ -d "$directory" ] && [ -w "$directory" ]; then
            zsh_completion_dir="$directory"
            break
          fi
          ;;
      esac
    done <<EOF
$zsh_paths
EOF
  fi
  if [ -n "$zsh_completion_dir" ]; then
    install_completion zsh "$zsh_completion_dir/_toggl"
  else
    install_completion zsh "$data_dir/zsh/site-functions/_toggl"
    echo "For zsh, place _toggl in a directory on your fpath and enable compinit. No writable default site-functions directory was found."
  fi
  echo "Open a new shell to load completions. Bash requires bash-completion; zsh requires compinit."
fi

case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to your PATH to run toggl." ;;
esac

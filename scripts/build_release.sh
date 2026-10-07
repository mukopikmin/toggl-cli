#!/bin/sh
set -eu
usage() { echo "usage: build_release.sh [--version] VERSION [--target linux-x64|darwin-arm64|windows-x64]" >&2; }
sha256_file() { if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'; elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'; else echo "required command not found: sha256sum or shasum" >&2; exit 1; fi; }
requested_version=""
selected_targets=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "Missing value for --version." >&2; exit 1; }
      requested_version=$2; shift 2 ;;
    --target)
      [ "$#" -ge 2 ] || { echo "Missing value for --target." >&2; exit 1; }
      case "$2" in linux-x64|darwin-arm64|windows-x64) ;; *) echo "Unknown target: $2. Expected one of: linux-x64, darwin-arm64, windows-x64" >&2; exit 1;; esac
      selected_targets="$selected_targets $2"; shift 2 ;;
    --*) echo "Unknown option: $1" >&2; usage; exit 1 ;;
    *) [ -z "$requested_version" ] || { echo "Unexpected argument: $1" >&2; usage; exit 1; }; requested_version=$1; shift ;;
  esac
done
[ -n "$requested_version" ] || { echo "Missing --version <version> for release archive names." >&2; exit 1; }
version="$requested_version"
if [ "$requested_version" = nightly ]; then
  commit_timestamp="$(git show -s --format=%ct HEAD)"
  commit_sha="$(git rev-parse --verify HEAD)"
  if commit_day="$(date -u -d "@$commit_timestamp" +%Y%m%d 2>/dev/null)"; then :; else commit_day="$(date -u -r "$commit_timestamp" +%Y%m%d)"; fi
  version="nightly-${commit_day}-$(printf '%s' "$commit_sha" | cut -c1-7)"
fi
if ! printf '%s\n' "$version" | grep -Eq '^([0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?|nightly-[0-9]{8}-[0-9a-f]{7})$'; then echo "Invalid version: $version" >&2; exit 1; fi
case " $selected_targets " in "  "|*" linux-x64 "*|*" darwin-arm64 "*) command -v tar >/dev/null 2>&1 || { echo "Missing required archive command: tar" >&2; exit 1; };; esac
case " $selected_targets " in "  "|*" windows-x64 "*) command -v zip >/dev/null 2>&1 || { echo "Missing required archive command: zip" >&2; exit 1; };; esac
mkdir -p dist
dist_dir="$(pwd)/dist"
current_stage=""
cleanup() { [ -z "$current_stage" ] || rm -rf "$current_stage"; }
trap cleanup EXIT HUP INT TERM
: > dist/checksums.txt
host_os="$(go env GOOS)"; host_arch="$(go env GOARCH)"
for spec in darwin-arm64:darwin:arm64 linux-x64:linux:amd64 windows-x64:windows:amd64; do
  target=${spec%%:*}
  if [ -n "$selected_targets" ]; then case " $selected_targets " in *" $target "*) ;; *) continue;; esac; fi
  rest=${spec#*:}; os=${rest%%:*}; arch=${rest##*:}
  root="toggl-cli-v${version}-${target}"; [ "$requested_version" = nightly ] && root="toggl-cli-nightly-${target}"
  rm -f "dist/$root.tar.gz" "dist/$root.tar.gz.sha256" "dist/$root.zip" "dist/$root.zip.sha256"
  current_stage="$(mktemp -d "${TMPDIR:-/tmp}/toggl-cli-release.XXXXXX")"
  mkdir -p "$current_stage/$root"
  binary=toggl; [ "$os" = windows ] && binary=toggl.exe
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$current_stage/$root/$binary" ./cmd/toggl
  cp README.md THIRD_PARTY_NOTICES.md "$current_stage/$root/"
  [ ! -f LICENSE ] || cp LICENSE "$current_stage/$root/LICENSE"
  if [ "$os" = "$host_os" ] && [ "$arch" = "$host_arch" ]; then test "$("$current_stage/$root/$binary" --version)" = "$version"; fi
  if [ "$os" = windows ]; then archive="$root.zip"; (cd "$current_stage" && zip -qr "$dist_dir/$archive" "$root"); else archive="$root.tar.gz"; tar -C "$current_stage" -czf "dist/$archive" "$root"; fi
  rm -rf "$current_stage"; current_stage=""
  checksum="$(sha256_file "dist/$archive")"
  printf '%s\n' "$checksum" > "dist/$archive.sha256"
  printf '%s  %s\n' "$checksum" "$archive" >> dist/checksums.txt
done

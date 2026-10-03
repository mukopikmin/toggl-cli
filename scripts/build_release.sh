#!/bin/sh
set -eu
requested_version="${1:?usage: build_release.sh VERSION}"
version="$requested_version"
if [ "$requested_version" = nightly ]; then
  commit_timestamp="$(git show -s --format=%ct HEAD)"
  commit_sha="$(git rev-parse --verify HEAD)"
  if commit_day="$(date -u -d "@$commit_timestamp" +%Y%m%d 2>/dev/null)"; then
    :
  else
    commit_day="$(date -u -r "$commit_timestamp" +%Y%m%d)"
  fi
  version="nightly-${commit_day}-$(printf '%s' "$commit_sha" | cut -c1-7)"
fi
if ! printf '%s\n' "$version" | grep -Eq '^([0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?|nightly-[0-9]{8}-[0-9a-f]{7})$'; then
  echo "invalid version: $version" >&2
  exit 1
fi
rm -rf dist
mkdir -p dist
for spec in linux-x64:linux:amd64 darwin-arm64:darwin:arm64 windows-x64:windows:amd64; do
  target=${spec%%:*}; rest=${spec#*:}; os=${rest%%:*}; arch=${rest##*:}
  root="toggl-cli-v${version}-${target}"
  [ "$requested_version" = nightly ] && root="toggl-cli-nightly-${target}"
  mkdir -p "dist/$root"
  binary=toggl; [ "$os" = windows ] && binary=toggl.exe
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" -o "dist/$root/$binary" ./cmd/toggl
  cp README.md "dist/$root/README.md"
  [ ! -f LICENSE ] || cp LICENSE "dist/$root/LICENSE"
  if [ "$os" = linux ] && [ "$arch" = amd64 ] && [ "$(uname -s):$(uname -m)" = "Linux:x86_64" ]; then
    test "$("dist/$root/$binary" --version)" = "$version"
  fi
  if [ "$os" = windows ]; then (cd dist && zip -qr "$root.zip" "$root"); else tar -C dist -czf "dist/$root.tar.gz" "$root"; fi
  rm -rf "dist/$root"
done
(cd dist && for f in *.tar.gz *.zip; do sha256sum "$f" | awk '{print $1}' > "$f.sha256"; done && sha256sum *.tar.gz *.zip > checksums.txt)

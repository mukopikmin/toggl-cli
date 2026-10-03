#!/bin/sh
set -eu
version="${1:?usage: build_release.sh VERSION}"
rm -rf dist
mkdir -p dist
for spec in linux-x64:linux:amd64 darwin-arm64:darwin:arm64 windows-x64:windows:amd64; do
  target=${spec%%:*}; rest=${spec#*:}; os=${rest%%:*}; arch=${rest##*:}
  root="toggl-cli-v${version}-${target}"; [ "$version" = nightly ] && root="toggl-cli-nightly-${target}"
  mkdir -p "dist/$root"
  binary=toggl; [ "$os" = windows ] && binary=toggl.exe
  GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" -o "dist/$root/$binary" ./cmd/toggl
  if [ "$os" = windows ]; then (cd dist && zip -qr "$root.zip" "$root"); else tar -C dist -czf "dist/$root.tar.gz" "$root"; fi
  rm -rf "dist/$root"
done
(cd dist && for f in *.tar.gz *.zip; do sha256sum "$f" | awk '{print $1}' > "$f.sha256"; done && sha256sum *.tar.gz *.zip > checksums.txt)

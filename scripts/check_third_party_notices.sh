#!/bin/sh
set -eu

test -s THIRD_PARTY_NOTICES.md || {
  echo "Missing or empty THIRD_PARTY_NOTICES.md" >&2
  exit 1
}

# Ignore Markdown blockquote markers and wrapping when comparing license texts.
normalize() {
  sed 's/^> *//' "$1" | awk '{ for (i = 1; i <= NF; i++) printf "%s ", $i }'
}
notices="$(normalize THIRD_PARTY_NOTICES.md)"

go mod download
modules="$(go list -m -f '{{if not .Main}}{{.Path}} {{.Dir}}{{end}}' all)"
printf '%s\n' "$modules" | while read -r module module_dir; do
  [ -n "$module" ] || continue
  case "$notices" in
    *"$module"*) ;;
    *) echo "Missing third-party notice for $module" >&2; exit 1 ;;
  esac

  license_file=""
  for name in LICENSE LICENSE.txt LICENSE.md COPYING; do
    if [ -s "$module_dir/$name" ]; then
      license_file="$module_dir/$name"
      break
    fi
  done
  if [ -z "$license_file" ]; then
    echo "Cannot find a license for $module in $module_dir" >&2
    exit 1
  fi
  license="$(normalize "$license_file")"
  if [ -z "$license" ]; then
    echo "Empty license for $module" >&2
    exit 1
  fi
  case "$notices" in
    *"$license"*) ;;
    *) echo "License text for $module differs from THIRD_PARTY_NOTICES.md" >&2; exit 1 ;;
  esac
done

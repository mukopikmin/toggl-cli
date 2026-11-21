#!/bin/sh
set -eu

cd "$(dirname "$0")/../.."

docker_arch=$(docker version --format '{{.Server.Arch}}')
work_dir=$(mktemp -d "${TMPDIR:-/tmp}/toggl-cli-e2e.XXXXXXXX")
container=${work_dir##*/}
image="toggl-cli-e2e:${container#toggl-cli-e2e.}"

cleanup() {
    docker container rm -f "$container" >/dev/null 2>&1 || true
    docker image rm "$image" >/dev/null 2>&1 || true
    rm -rf -- "$work_dir"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

mkdir "$work_dir/rootfs"
export CGO_ENABLED=0 GOOS=linux GOARCH="$docker_arch"
go build -o "$work_dir/rootfs/toggl" ./cmd/toggl
go test -c -tags=e2e -o "$work_dir/rootfs/cli.test" ./test/e2e
chmod 755 "$work_dir/rootfs/toggl" "$work_dir/rootfs/cli.test"

# Build a scratch image from the two static binaries.
tar -C "$work_dir/rootfs" -cf "$work_dir/rootfs.tar" toggl cli.test
docker import "$work_dir/rootfs.tar" "$image" >/dev/null
docker run --rm --pull never --name "$container" --network none \
    --add-host api.track.toggl.com:127.0.0.1 --user 65532:65532 --cap-drop ALL \
    --read-only --tmpfs /tmp:nosuid,nodev,noexec,mode=1777 \
    "$image" /cli.test -test.v -test.timeout=1m

.DEFAULT_GOAL := help

VERSION ?=
TARGET ?=
ARGS ?=

.PHONY: help fmt check-fmt check-shell check-notices vet test check-build check build run release

help:
	@printf '%s\n' \
	  'make check          Run all CI checks' \
	  'make fmt            Format Go source files' \
	  'make build          Build out/toggl' \
	  'make run ARGS="..."  Run the CLI from source' \
	  'make test           Run all tests' \
	  'make vet            Run static analysis' \
	  'make release VERSION=0.1.0 [TARGET=linux-x64]' \
	  '                    Build release archives in dist/'

fmt:
	gofmt -w .

check-fmt:
	@test -z "$$(gofmt -l .)"

check-shell:
	sh -n install.sh

check-notices:
	sh scripts/check_third_party_notices.sh

vet:
	go vet ./...

test:
	go test ./...

check-build:
	go build ./...

check: check-fmt check-shell check-notices vet test check-build

build:
	go build -o ./out/toggl ./cmd/toggl

run:
	go run ./cmd/toggl $(ARGS)

release:
	@test -n "$(VERSION)" || { echo 'Usage: make release VERSION=0.1.0 [TARGET=linux-x64]' >&2; exit 1; }
	./scripts/build_release.sh --version "$(VERSION)" $(if $(TARGET),--target "$(TARGET)")

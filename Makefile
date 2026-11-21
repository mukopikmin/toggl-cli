.DEFAULT_GOAL := help

VERSION ?=
TARGET ?=
ARGS ?=
VULNDB ?= https://vuln.go.dev

.PHONY: help fmt check-fmt check-shell check-notices check-vuln vet test test-e2e check-build check build run release

help:
	@printf '%s\n' \
	  'make check          Run the standard checks' \
	  'make fmt            Format Go source files' \
	  'make build          Build out/toggl' \
	  'make run ARGS="..."  Run the CLI from source' \
	  'make test           Run Go tests' \
	  'make test-e2e       Run the CLI E2E tests in Docker' \
	  '                    Set TOGGL_E2E_BINARY to test a prebuilt binary' \
	  'make vet            Run static analysis' \
	  'make check-vuln     Scan for known Go vulnerabilities' \
	  'make release VERSION=0.1.0 [TARGET=linux-x64]' \
	  '                    Build release archives in dist/'

fmt:
	gofmt -w .

check-fmt:
	@test -z "$$(gofmt -l .)"

check-shell:
	sh -n install.sh
	sh -n scripts/install-completions.sh
	sh -n scripts/build_release.sh
	sh -n test/e2e/run.sh
	bash -n internal/command/completions/bash

check-notices:
	sh scripts/check_third_party_notices.sh

vet:
	go vet ./...

test:
	go test ./...

test-e2e:
	sh test/e2e/run.sh

check-build:
	go build ./...

check-vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 -db "$(VULNDB)" ./...

check: check-fmt check-shell check-notices vet test check-build check-vuln

build:
	go build -o ./out/toggl ./cmd/toggl

run:
	go run ./cmd/toggl $(ARGS)

release:
	@test -n "$(VERSION)" || { echo 'Usage: make release VERSION=0.1.0 [TARGET=linux-x64]' >&2; exit 1; }
	./scripts/build_release.sh --version "$(VERSION)" $(if $(TARGET),--target "$(TARGET)")

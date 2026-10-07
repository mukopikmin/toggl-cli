# toggl-cli

A Go CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values, JSON, or bordered terminal tables.

Configuration is parsed with `github.com/BurntSushi/toml`, which is distributed
under the permissive MIT license. Third-party license texts are recorded in
`THIRD_PARTY_NOTICES.md` and included in release archives.

## Requirements

- A Toggl Track API token and workspace ID
- Go 1.23 or later (only when building from source)

## Installation

Install the latest Linux x64 or macOS arm64 release:

```sh
curl -fsSL https://raw.githubusercontent.com/mukopikmin/toggl-cli/main/install.sh | sh
```

The binary is installed as `$HOME/.local/bin/toggl`. Add `--nightly` to install
the latest tested nightly. Windows x64 archives are available on the release
page.

Build and install from source with:

```sh
go install github.com/mukopikmin/toggl-cli/cmd/toggl@latest
# or from a checkout
go build -o ./out/toggl ./cmd/toggl
```

## Configuration

Run `toggl init`, or create `~/.config/toggl-cli/config.toml`:

```toml
workspace = "your_workspace_id"
token = "your_api_token"
timezone = "Asia/Tokyo"

[projects."123456"]
display_name = "Client A"
display_order = 10
hidden = false
```

The timezone defaults to the system timezone when omitted. Project settings are
optional. `display_name` changes output, `display_order` sorts configured
projects first, and `hidden` excludes a project. The token is never shown by
`toggl config`. Protect the file with `chmod 600`.

To migrate an old `~/.toggl_config`, run this from a checkout:

```sh
go run ./cmd/migrate-config
```

## Command compatibility

The Go implementation preserves the command surface, defaults, output formats,
and successful/error exit codes of the previous implementation:

| Command | Options / arguments | Output |
| --- | --- | --- |
| `summary` | `<start-date> <end-date>` or `--days N`; `-f/--format csv\|json\|table`, `-s/--separator`, `--no-project`, `--no-date`, `--clipboard` | Tab-separated CSV by default; JSON summary or bordered table |
| `time-entry list` | `<start-day> <end-day>`; `-f/--format`, `-s/--separator` | Entries as CSV, JSON, or table |
| `project list` | `-f/--format` | Visible ordered project names, JSON, or table |
| `project sync` | none | Adds missing active projects to the TOML file |
| `project reorder` | none | Interactively changes and saves visible-project order |
| `config` | `-f/--format` | Non-sensitive settings as `KEY=VALUE`, JSON, or table |
| `init` | none | Creates a mode-`0600` configuration interactively |
| `update` | `--channel stable\|nightly` | Checks, verifies, and installs a release update |
| `--help`, `-h` | none | Help text |
| `--version` | none | Build version |

Date ranges are inclusive. Invalid usage, configuration, API, clipboard, and
I/O errors exit with status 1; successful commands exit with status 0.

`project reorder` requires an interactive terminal. Use `j`/`k` or the arrow
keys to select, Space to pick or drop, uppercase `J`/`K` to move, Enter to save,
and `q` or Escape to cancel. The screen shows one project per line, with `>`
marking the selected project and `*` marking a picked project.

`toggl update` is available in compiled release binaries. It selects the stable
or nightly channel, verifies the downloaded archive's SHA-256 checksum and
embedded version, and only then replaces the current executable. Development
builds such as `0.0.0-dev` must be updated by building or installing again.

## Architecture

- `cmd/toggl/main.go`: argument entry point and dependency assembly.
- `internal/command/`: CLI interpretation and output formatting.
- `internal/model/`: API-independent models and pure aggregation.
- `internal/toggl/`: HTTP client, private DTOs, and domain conversion.
- `internal/config/`: configuration loading and validation.

## Development

```sh
go run ./cmd/toggl -- summary 2026-06-01 2026-06-15
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

Build all release archives with `./scripts/build_release.sh --version 0.1.0`.
The positional form `./scripts/build_release.sh 0.1.0` remains available, and
`--target linux-x64` (or `darwin-arm64` / `windows-x64`) limits the build to one
or more targets. CI uses the same formatting, shell syntax, static-analysis,
test, and build checks.

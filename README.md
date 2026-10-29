# toggl-cli

A Go CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values, JSON, or bordered terminal tables.

Configuration is parsed with `github.com/BurntSushi/toml`, which is distributed
under the permissive MIT license. Third-party license texts are recorded in
`THIRD_PARTY_NOTICES.md` and included in release archives.

## Requirements

- A Toggl Track API token and workspace ID
- Go 1.27.3 or later, as required by `go.mod` (only when building from source)

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
make build
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

`toggl init` requires a workspace ID and API token. Enter a blank timezone to
use `Asia/Tokyo`. If input ends before all three prompts are answered, no
configuration file is created. The `projects` configuration must be a TOML
table, such as `[projects.123456]` above; an empty table (`projects = {}`) is
also accepted.

To migrate an old `~/.toggl_config`, run this from a checkout:

```sh
go run ./cmd/migrate-config
```

## Command compatibility

The Go implementation preserves the command surface, defaults, output formats,
and successful/error exit codes of the previous implementation:

| Command | Options / arguments | Output |
| --- | --- | --- |
| `summary` | `<start-date> <end-date>`, `--days N`, or `--month MONTH`; `-f/--format csv\|json\|table`, `-s/--separator`, `--no-project`, `--no-date`, `--clipboard` | Tab-separated CSV by default; JSON summary or bordered table |
| `time-entry list` | `<start-day> <end-day>`; `-f/--format`, `-s/--separator` | Entries as CSV, JSON, or table |
| `project list` | `-f/--format` | Visible ordered project names, JSON, or table |
| `client list` | `-f/--format`, `-s/--separator` | Client IDs and names as CSV/TSV or JSON; names as a table |
| `project sync` | none | Adds missing active projects to the TOML file |
| `project reorder` | none | Interactively changes and saves visible-project order |
| `config` | `-f/--format` | Non-sensitive settings as `KEY=VALUE`, JSON, or table |
| `init` | none | Creates a mode-`0600` configuration interactively |
| `update` | `--channel stable\|nightly` | Checks, verifies, and installs a release update |
| `--help`, `-h` | none | Help text |
| `--version` | none | Build version |

Date ranges are inclusive. Invalid usage, configuration, API, clipboard, and
I/O errors exit with status 1; successful commands exit with status 0.

Use `toggl summary --month 2026-10` to summarize October 1 through October 31,
2026. Months accept `YYYY-MM`, `YYYYMM`, and `YYYY/MM`, with a four-digit year and
two-digit month: `2026-10`, `202610`, and `2026/10` select the same month.
The range includes the entire month in the configured time zone (or the system
time zone when unset), even for the current month. `--month`, `--days`, and
explicit start/end dates are mutually exclusive.
The `--month` option is available only for `summary`.

Short options accept attached values, such as `-fjson`, `-d7`, and `-s,`.
Use `--` to end option parsing. Boolean flags such as `--no-date` do not accept
values. For a separator starting with a dash, use `--separator=<text>`.
The `--days N` option includes exactly N calendar days ending today in the
configured timezone (or the system timezone when omitted). For example,
`--days 1` includes only today, and `--days 7` includes today and the previous
six days. The value must be a positive integer written using digits; zero is
invalid. `time-entry list` days must exist in the current month. Invalid command
usage is reported before configuration is loaded. An empty
`project list --format json` or `client list --format json` result is `[]`.

`toggl client list` fetches clients from the configured workspace in API order.
The default output is TSV with `id` and `name` headers. The `csv` format uses
`--separator` to select the delimiter, with tab as the default. Fields containing
the delimiter, quotes, or newlines are quoted. Empty CSV/TSV output contains only
the header. The separator must not be empty and cannot be used with table format.
This command does not change the configuration file.

```sh
toggl client list                   # TSV: id and name columns
toggl client list --separator ,     # CSV: id and name columns
toggl client list --format json     # Objects with id and name
toggl client list --format table    # Bordered table of names
```

`project reorder` requires an interactive terminal. Use `j`/`k` or the arrow
keys to select, Space to pick or drop, uppercase `J`/`K` to move, Enter to save,
and `q` or Escape to cancel. The screen shows one project per line, with `>`
marking the selected project and `*` marking a picked project.

`toggl update` is available in compiled release binaries. It selects the stable
or nightly channel, verifies the downloaded archive's SHA-256 checksum and
embedded version, and only then replaces the current executable. Development
builds such as `0.0.0-dev` must be updated by building or installing again.
On Windows, the executable is replaced after the current process exits.

## Architecture

- `cmd/toggl/main.go`: argument entry point and dependency assembly.
- `internal/command/`: CLI interpretation and output formatting. `options.go`
  handles shared option parsing; `update_install.go` handles update installation.
- `internal/model/`: API-independent models and pure aggregation.
- `internal/toggl/`: HTTP client, private DTOs, and domain conversion.
- `internal/config/`: configuration loading and validation.

## Development

Install Make and the Go version required by `go.mod`. From a checkout:

```sh
make                     # Show available commands
make run ARGS="summary 2026-06-01 2026-06-15"
make fmt                 # Format Go source files
make check               # Run all CI checks
make build               # Build ./out/toggl
```

`make check` runs `check-fmt`, `check-shell`, `check-notices`, `vet`, `test`,
`check-build`, and `check-vuln`. Each target can also be run separately. CI calls
these same targets for formatting, installer shell syntax, third-party notices,
static analysis, tests, package builds, and known vulnerabilities.

The `Test` workflow runs for pull requests, pushes to `main`, and manual dispatches.
Feature-branch pushes use the pull request run to avoid duplicate checks. Successful
checks on `main` continue to trigger the Nightly workflow.

`make check-vuln` runs `govulncheck` v1.9.0 for the current platform, checking the
Go standard library and dependencies for known vulnerabilities reachable from
the code. It downloads the scanner and queries the Go vulnerability database,
so network access is required. Vulnerability findings fail the check.
Set `VULNDB` to use another database URL or a local database with a `file://` URL.

Build all release archives with `make release VERSION=0.1.0`, or use
`VERSION=nightly` for nightly archives. Add `TARGET=linux-x64` (or `darwin-arm64`
/ `windows-x64`) to build a single target. Archives are written to `dist/`.
The Make target delegates to `scripts/build_release.sh`, whose direct invocation
still supports `--version` and repeated `--target` options. Release workflows
use the same Make target.

Dependabot checks Go modules and GitHub Actions weekly, groups updates by
ecosystem, and waits seven days after new versions are released. CI and release
workflows select Go from `go.mod`, so dependency updates can also raise the
required Go version. The notices check verifies that every dependency module
and its license text are covered by `THIRD_PARTY_NOTICES.md`; update the notices
when adding a dependency or when an upstream license changes.

## Stable releases

Stable releases follow the same tag-driven flow as `mukopikmin/sadoku`.
Use the `toggl-cli-release` skill to review changes, propose a semantic version,
and approve the exact tested `main` commit, notes range, and omitted pull requests.
Record the approved comparison tag in the annotated tag's `Previous-Stable-Tag`
trailer, for example:

```sh
git tag --annotate v0.5.2 <approved-full-sha> \
  --message "Release v0.5.2" \
  --message "Previous-Stable-Tag: v0.5.0"
```

Pushing that annotated `vMAJOR.MINOR.PATCH` tag starts the `Release` workflow and
authorizes automatic build and publication; there is no separate preparation
run or release-plan hash approval.

Actions validates that the unchanged tag points to the current tip of `main`,
that the exact commit passed the `Test` push workflow, and that the version is
newer than the previous stable release. For the first release,
`release-notes-baseline` must point to the repository root commit. Drafts,
prereleases, and nightly releases are excluded from the notes baseline. Each
validation and notes step reads the approved baseline from the tag trailer and
compares it with a fresh GitHub lookup. An empty stable-release lookup with other
stable version tags on the remote fails instead of implying a first release.
Missing, duplicate, or invalid trailers also fail. Validation JSON includes the
release candidates, selection reason, remote stable tags, and returned CI runs
(ID, workflow, branch, event, status, conclusion, SHA, and URL) for diagnosis.

The build job creates and attests the three platform archives. The publish job
verifies their names, checksums, and provenance, generates notes from the previous
stable tag, and revalidates the target immediately before creating the latest
stable GitHub release. It then verifies the published notes, assets, and tag.
Failed runs leave the stable tag intact; never move or recreate it.

The workflow uses `go run ./scripts/release check --version 0.1.0` for validation
and `go run ./scripts/release notes --version 0.1.0 --output release-notes.json`
for notes. These commands require Git, an authenticated GitHub CLI, and a checkout
with the annotated release tag (including its `Previous-Stable-Tag` trailer) and
comparison tag available. Notes generation uses GitHub's
generate-notes API. Nightly publication continues independently after successful
tests on `main`.

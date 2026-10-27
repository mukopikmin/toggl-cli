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
| `summary` | `<start-date> <end-date>`, `--days N`, or `--template NAME`; `-f/--format csv\|json\|table`, `-s/--separator`, `--no-project`/`--project`, `--no-date`/`--date`, `--clipboard`/`--no-clipboard` | Tab-separated CSV by default; JSON summary or bordered table |
| `template add` | `<name> [<start-date> <end-date> or --days N]`; summary output options, `--force` | Stores a summary template without running it |
| `template list` | none | Template names in alphabetical order |
| `template show` | `<name>` | Template settings as TOML |
| `template delete` | `<name>` | Deletes the named template |
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

Short options accept attached values, such as `-fjson`, `-d7`, and `-s,`.
Use `--` to end option parsing. Boolean flags such as `--no-date` do not accept
values. For a separator starting with a dash, use `--separator=<text>`.
The `--days` value must be a non-negative integer written using digits, and
`time-entry list` days must exist in the current month. Invalid command usage
is reported before configuration is loaded. An empty
`project list --format json` result is `[]`.

`project reorder` requires an interactive terminal. Use `j`/`k` or the arrow
keys to select, Space to pick or drop, uppercase `J`/`K` to move, Enter to save,
and `q` or Escape to cancel. The screen shows one project per line, with `>`
marking the selected project and `*` marking a picked project.

`toggl update` is available in compiled release binaries. It selects the stable
or nightly channel, verifies the downloaded archive's SHA-256 checksum and
embedded version, and only then replaces the current executable. Development
builds such as `0.0.0-dev` must be updated by building or installing again.
On Windows, the executable is replaced after the current process exits.

## Summary templates

Store frequently used summary settings, then run them by name:

```sh
# Store a rolling seven-day summary, including today; no API request is made.
toggl template add weekly --days 6 --format table
toggl template list
toggl template show weekly
toggl summary --template weekly

# Override settings for this execution only.
toggl summary --template weekly --days 13 --format csv
toggl summary --template weekly 2026-10-01 2026-10-31

# Store only output settings and supply a period when running the summary.
toggl template add spreadsheet --no-project --no-date --clipboard
toggl summary --template spreadsheet 2026-10-01 2026-10-31
toggl summary --template spreadsheet --days 0 --project --date --no-clipboard

# Replace all settings of an existing template, or delete it.
toggl template add weekly --days 6 --format json --force
toggl template delete weekly
```

Templates are stored in `~/.config/toggl-cli/templates.toml`, separately from
credentials and project settings. Management commands do not require `toggl init`
or API access. A missing templates file is treated as an empty collection.
Names must start with an ASCII letter or digit and contain only ASCII letters,
digits, underscores, or hyphens. Names are case-sensitive. Adding an existing
name fails unless `--force` is given. Unknown names and invalid template files
produce errors; an invalid file is never overwritten by a management command.

The file uses named TOML tables, for example:

```toml
[templates.weekly]
days = 6
format = "table"
no_project = false
no_date = false
clipboard = false

[templates.spreadsheet]
format = "csv"
separator = "\t"
no_project = true
no_date = true
clipboard = true
```

Periods can use `days`, or both `start_date` and `end_date` in `YYYY-MM-DD`
format, or be omitted. `days = 0` means today; `days = 6` means six days ago
through today (seven inclusive dates). Relative dates are calculated each time
the summary runs, using the configured timezone or system timezone. Fixed dates
remain fixed. A period must be available when executing a summary.

Explicit command-line settings override the template, followed by the existing
defaults (CSV, tab separator, project column and date header, no clipboard).
A command-line period replaces the saved period entirely. Supplying both a date
range and `--days` on the command line is still an error. Use `--project`,
`--date`, and `--no-clipboard` to reverse saved boolean settings; opposing flags
use the last value specified. Boolean flags do not accept `=true` or `=false`.
Merged settings are validated before API access, including the restriction on
using `--separator`, `--no-project`, or `--no-date` with table output. Overrides
do not modify the stored template. `template add --force` replaces all settings,
rather than merging them with the previous template.

## Architecture

- `cmd/toggl/main.go`: argument entry point and dependency assembly.
- `internal/command/`: CLI interpretation and output formatting. `options.go`
  handles shared option parsing; `update_install.go` handles update installation.
- `internal/model/`: API-independent models and pure aggregation.
- `internal/template/`: validated TOML storage for named summary settings.
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

# toggl-cli

A Go CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values, JSON, or bordered terminal tables.

Configuration is parsed with `github.com/BurntSushi/toml`, which is distributed
under the permissive MIT license. Third-party license texts are recorded in
`THIRD_PARTY_NOTICES.md` and included in release archives.

## Requirements

- A Toggl Track API token and workspace ID
- Go 1.27.4 or later, as required by `go.mod` (only when building from source)

## Installation

Installation and self-update verify the public Sigstore provenance bundle for the release commit and workflow. No GitHub account or GitHub CLI is required. They use Cosign v3 on PATH when available, otherwise they temporarily download a SHA-256-pinned Cosign v3.0.6 binary (about 135 MB on Linux x64). The public Sigstore trust root is pinned. Verification requires network access to GitHub. The installer supports the existing v0.5.2 public attestation. Releases without a verification bundle require a source build.

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

## Shell completion

The installer places completion files automatically, without editing shell
startup files or using `sudo`:

| Shell | Installation location | Prerequisite |
| --- | --- | --- |
| bash | `${BASH_COMPLETION_USER_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion}/completions/toggl` | bash-completion must already be loaded |
| fish | `${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions/toggl.fish` | fish loads this directory automatically |
| zsh | The first existing writable `site-functions` directory in zsh's default `fpath` | `compinit` must already be enabled |

Open a new shell after installation. Completion covers commands, subcommands,
supported options, output formats (`tsv`, `csv`, `json`, `table`), update channels, and
completion shell names. It does not require configuration or contact the Toggl
API. Dates and arbitrary option values are left to the user.

When no writable default zsh `site-functions` directory is available, the
installer saves `_toggl` to
`${XDG_DATA_HOME:-$HOME/.local/share}/zsh/site-functions/_toggl` and prints a
notice. Move that file into an existing directory on your `fpath`, or add its
directory to `fpath` before running `compinit`. A shell without its completion
framework enabled still needs that framework configured; installing a CLI cannot
enable it in the parent shell.

A separate `install-completions.sh` is included in each release archive. The
Linux/macOS installer and `toggl update` both run this script. After extracting
a release manually, run it from the release directory:

```sh
sh ./install-completions.sh
```

From a source checkout, with `toggl` already on your PATH:

```sh
sh scripts/install-completions.sh internal/command/completions
```

The script installs all three shells, prints loading instructions, and can be
rerun to replace completion files. It returns a nonzero status if installation
fails. Initial installation and updates report completion failures while
preserving a successful binary installation. Windows updates replace only the
binary; users of POSIX shells on Windows can run the script manually.

To print a script instead of installing it:

```sh
toggl completion bash
toggl completion zsh
toggl completion fish
```

For example, load completion in the current bash or zsh session with
`source <(toggl completion bash)` or `source <(toggl completion zsh)` (after
`compinit` in zsh). In fish, use `toggl completion fish | source`. For persistent
completion, save the output in the corresponding directory above. Release
archives also include all three scripts in `completions/`. Completion scripts
ask the installed binary for candidates, so `toggl update` keeps the candidates
current. On Linux/macOS, updates also refresh the scripts using the installer
from the checksum-verified release archive. Releases without the installer remain
installable and skip completion installation. Upgrading from a version that
predates this update behavior requires running the script manually once, or
rerunning the main installer.

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

## Commands

Use `-f/--format tsv|csv|json|table` to select an output format. TSV uses tabs
and CSV uses commas; both delimiters are fixed. Fields containing the delimiter,
quotes, or line breaks are quoted, and embedded quotes are doubled.

| Command | Options / arguments | Output |
| --- | --- | --- |
| `summary` | `<start-date> <end-date>`, `--days N`, or `--template NAME`; `-f/--format`, `--watch`, `--no-project`/`--project`, `--no-date`/`--date`, `--daily-total`, `--clipboard`/`--no-clipboard` | TSV by default; CSV, JSON summary, or bordered table |
| `template add` | `<name> [<start-date> <end-date> or --days N]`; summary output options, `--force` | Stores a summary template without running it |
| `template list` | none | Template names in alphabetical order |
| `template show` | `<name>` | Template settings as TOML |
| `template delete` | `<name>` | Deletes the named template |
| `time-entry list` | `<start-day> <end-day>`; `-f/--format`, `--watch` | Entries as TSV by default, CSV, JSON, or table |
| `project list` | `-f/--format`, `--watch` | Visible ordered project names as TSV or CSV, JSON, or table |
| `client list` | `-f/--format`, `--watch` | Client IDs and names as TSV by default, CSV or JSON; names as a table |
| `project sync` | none | Adds missing active projects to the TOML file |
| `project reorder` | none | Interactively changes and saves visible-project order |
| `config` | `-f/--format`, `--watch` | Non-sensitive settings as two-column TSV or CSV with a `Setting`/`Value` header, JSON, or table |
| `init` | none | Creates a mode-`0600` configuration interactively |
| `update` | `--channel stable\|nightly` | Checks, verifies, and installs a release update |
| `completion` | `bash\|zsh\|fish` | Prints a shell completion script without loading configuration |
| `--help`, `-h` | none | Help text |
| `--version` | none | Build version |

Date ranges are inclusive. Invalid usage, configuration, API, clipboard, and
I/O errors exit with status 1; successful commands exit with status 0.

Short options accept attached values, such as `-fjson`, `-ftsv`, and `-d7`.
Use `--` to end option parsing. Boolean flags such as `--no-date` do not accept
values. CSV now uses commas and TSV uses tabs. The `-s/--separator` option has
been removed; replace `--separator ,` with `--format csv`.
The `--days N` option includes exactly N calendar days ending today in the
configured timezone (or the system timezone when omitted). For example,
`--days 1` includes only today, and `--days 7` includes today and the previous
six days. The value must be a positive integer written using digits; zero is
invalid. `time-entry list` days must exist in the current month. Invalid command
usage is reported before configuration is loaded. An empty
`project list --format json` or `client list --format json` result is `[]`.

Table output uses the terminal width at the time of rendering. If the full
headings are too wide, it uses shorter headings when they reduce the table width.
In a `summary` covering a single year, shorter date headings use `MM-DD`.
When space is tighter, the headings can show just the day of the month, such as
`9` or `10`. For a range spanning multiple months, the first displayed date of
each month includes the month and, if the range spans multiple years, the year.
If shorter headings still leave the table too wide, `summary` and
`project list` shorten project names with `…`. Other data cells keep their full
contents, so the table can still exceed the terminal width. Piped and
redirected output uses the full headings and project names.

Use `--watch` with table format in a terminal to redraw the table after a
resize. It reads the data once and displays the portion that fits on screen.
Press Ctrl+C to leave the live view and print the entire table with full project
names. For `summary`, `--clipboard` also preserves full project names.
Pass `--watch` when running a summary, including when the table format comes
from a saved template.

```sh
toggl summary --days 7 --format table --watch
```

`toggl client list` fetches clients from the configured workspace in API order.
The default output is TSV with `id` and `name` headers. CSV uses commas and TSV
uses tabs. Fields containing the delimiter, quotes, or newlines are quoted.
Empty CSV/TSV output contains only the header. This command does not change the
configuration file.

```sh
toggl client list                   # TSV: id and name columns
toggl client list --format csv      # CSV: id and name columns
toggl client list --format json     # Objects with id and name
toggl client list --format table    # Bordered table of names
```

Add `--daily-total` to `summary` to append a `Total` row to CSV, TSV, or table output:

```sh
toggl summary 2026-06-01 2026-06-15 --daily-total
toggl summary --days 7 --daily-total --format table
```

Totals are in minutes and include only the projects shown in the output. Hidden
projects, projects absent from the visible project list, and entries without a
project are excluded. Dates follow the configured timezone and the entry's start
date, as in the project rows. Days without time entries total `0`. Totals are
computed before rounding and displayed to at most two decimal places. The flag
works with `--no-project`, `--no-date`, and `--clipboard` where
those options are supported; JSON does not support `--daily-total`.

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
toggl template add weekly --days 7 --format table
toggl template list
toggl template show weekly
toggl summary --template weekly

# Override settings for this execution only.
toggl summary --template weekly --days 14 --format csv
toggl summary --template weekly 2026-10-01 2026-10-31

# Store only output settings and supply a period when running the summary.
toggl template add spreadsheet --no-project --no-date --clipboard
toggl summary --template spreadsheet 2026-10-01 2026-10-31
toggl summary --template spreadsheet --days 1 --project --date --no-clipboard

# Replace all settings of an existing template, or delete it.
toggl template add weekly --days 7 --format json --force
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
days = 7
format = "table"
no_project = false
no_date = false
clipboard = false

[templates.spreadsheet]
format = "csv"
no_project = true
no_date = true
clipboard = true
```

Periods can use `days`, or both `start_date` and `end_date` in `YYYY-MM-DD`
format, or be omitted. `days = 1` means today; `days = 7` means seven calendar days
ending today (seven inclusive dates). Relative dates are calculated each time
the summary runs, using the configured timezone or system timezone. Fixed dates
remain fixed. A period must be available when executing a summary.

Explicit command-line settings override the template, followed by the existing
defaults (TSV, project column and date header, no clipboard).
A command-line period replaces the saved period entirely. Supplying both a date
range and `--days` on the command line is still an error. Use `--project`,
`--date`, and `--no-clipboard` to reverse saved boolean settings; opposing flags
use the last value specified. Boolean flags do not accept `=true` or `=false`.
Merged settings are validated before API access, including the restriction on
using `--no-project` or `--no-date` with table output. Previously saved
`separator` template settings are ignored; the selected format determines the
delimiter. Overrides do not modify the stored template. `template add --force`
replaces all settings, rather than merging them with the previous template.

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

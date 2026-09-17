# toggl-cli

A Deno CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values, JSON, or bordered terminal tables.

## Requirements

- A Toggl Track API token
- A workspace ID
- Deno 2.8 or later (when running from source)

## Installation

Install the latest release on Linux x64 or macOS arm64:

```sh
curl -fsSL https://raw.githubusercontent.com/mukopikmin/toggl-cli/main/install.sh | sh
```

The binary is installed to `$HOME/.local/bin/toggl`; make sure that directory is
in your `PATH`. To install the latest tested nightly build, add `--nightly`:

```sh
curl -fsSL https://raw.githubusercontent.com/mukopikmin/toggl-cli/main/install.sh | sh -s -- --nightly
```

Windows users can download the `windows-x64` release archive and place
`toggl.exe` in a directory on `PATH`.

## Configuration

Create `~/.config/toggl-cli/config.toml` interactively:

```sh
toggl init
```

This asks for a workspace ID, API token, and timezone. The workspace ID and API
token are required; interactive input retries empty values, while incomplete
non-interactive input does not create the file. On POSIX systems, the file is
created with permissions set to `0600`. The API token is not printed after
entry. You can also create it manually:

```toml
workspace = "your_workspace_id"
token = "your_api_token"
timezone = "Asia/Tokyo"

[projects."123456"]
display_name = "Client A"
display_order = 10
hidden = false
```

`timezone` is optional and defaults to the execution environment's timezone.
Project settings are also optional: `display_name` changes the displayed name,
`display_order` sorts configured projects first in ascending order, and `hidden`
excludes a project from project lists and CSV summaries.

The config contains credentials, so restrict access to it:

```sh
chmod 600 ~/.config/toggl-cli/config.toml
```

To import an old `~/.toggl_config` file, run `deno task migrate-config` from a
repository checkout.

## Usage

```text
toggl summary <start-date> <end-date> [options]
toggl summary --days <days> [options]
toggl time-entry list <start-day> <end-day> [options]
toggl project list [options]
toggl project reorder
toggl project sync
toggl config [options]
toggl init
toggl update [--channel stable|nightly]
```

Run `toggl --help` for the complete option list.

### Examples

```sh
# Summarize an inclusive date range as tab-separated values.
toggl summary 2026-06-01 2026-06-15

# Summarize the previous seven days and today as JSON.
toggl summary --days 7 --format json

# Produce comma-separated values without project or date headings.
toggl summary 2026-06-01 2026-06-15 -s "," --no-project --no-date

# Print a summary and copy it to the clipboard.
toggl summary 2026-06-01 2026-06-15 --clipboard

# List entries from the 1st through the 15th of the current month.
toggl time-entry list 1 15

# List projects in a bordered table.
toggl project list --format table

# Reorder visible projects interactively.
toggl project reorder
```

Summary dates and time-entry day ranges are inclusive. CSV output uses tabs by
default; `--separator` (`-s`) changes the delimiter, and `--format json`
(`-f json`) selects JSON output.

`toggl update` updates a compiled installation in place and verifies its
checksum and version. Linux x64 and macOS arm64 are supported; source-based,
Windows, and other installations must be updated manually.

## Development

Run commands from a checkout with `deno task run --`, for example:

```sh
deno task run -- summary 2026-06-01 2026-06-15
```

Install or build a standalone binary:

```sh
deno task install --version 0.1.0
deno task compile --version 0.1.0
```

Run the same checks as CI:

```sh
deno fmt --check
sh -n install.sh
deno task check
deno task test
deno task compile --output /tmp/toggl-cli
```

Build release archives for all supported platforms with
`deno task dist --version 0.1.0`, or add `--target linux-x64` to build one
target. Stable releases are prepared with `release_prepare.yml` and published
from the approved candidate with `release.yml`; successful `main` builds publish
the moving `nightly` release automatically.

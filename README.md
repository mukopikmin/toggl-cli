# toggl-cli

A Deno CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values or JSON.

## Requirements

- A Toggl Track API token
- The ID of the target workspace

Building from source additionally requires Deno 2.8 or later.

## Installation

On Linux x64 and macOS arm64, install the latest release binary to
`$HOME/.local/bin/toggl` without cloning the repository:

```sh
curl -fsSL https://raw.githubusercontent.com/mukopikmin/toggl-cli/main/install.sh | sh
```

To install the latest tested nightly build instead:

```sh
curl -fsSL https://raw.githubusercontent.com/mukopikmin/toggl-cli/main/install.sh | sh -s -- --nightly
```

Make sure `$HOME/.local/bin` is included in your `PATH`.

To build and install from source, clone the repository and run:

```sh
deno task install --version 0.1.0
```

On Windows, download the `windows-x64` release archive, extract `toggl.exe`, and
place it in a directory included in your `PATH`.

## Configuration

Create a config file:

```sh
toggl init
```

This asks for your workspace ID, API token, and timezone, then creates
`~/.config/toggl-cli/config.toml` if it does not already exist. The API token is
not printed back to the terminal after entry. You can also create the file
manually:

```toml
workspace = "your_workspace_id"
token = "your_api_token"
timezone = "Asia/Tokyo"
```

Optional per-project settings can be configured with the `projects` table:

```toml
[projects."123456"]
display_name = "Client A"
hidden = false
display_order = 10

[projects."789012"]
hidden = true
display_order = 20
```

Display names are used when rendering project lists and summary CSV output. When
`display_name` is omitted, the Toggl project name is used. When `hidden` is
omitted, it defaults to `false`. Hidden projects are excluded from `projects`
output and summary CSV output. The optional `display_order` setting controls the
order of visible projects in `projects` output and summary CSV rows. Projects
with `display_order` are shown first in ascending numeric order, and projects
without `display_order` keep their Toggl API order after the ordered projects.

The optional `timezone` setting is used to calculate the Toggl time entry query
range. When it is omitted, the CLI uses the execution environment's timezone.

To migrate an old `~/.toggl_config` file, run:

```sh
deno task migrate-config
```

You can find your API token in your Toggl Track profile settings. Because the
configuration file contains credentials, restrict its permissions so that other
users cannot read it:

```sh
chmod 600 ~/.config/toggl-cli/config.toml
```

## Usage

Show command-line help:

```sh
deno task run -- --help
```

### Commands

| Command                              | Description                                            |
| ------------------------------------ | ------------------------------------------------------ |
| `summary <start-date> <end-date>`    | Aggregate time entries for a date range.               |
| `projects`                           | List active, visible projects.                         |
| `projects sync`                      | Add missing active projects to the configuration file. |
| `init`                               | Create the configuration file.                         |
| `update [--channel stable\|nightly]` | Update the installed compiled binary.                  |

### Options

| Option                     | Description                                                     |
| -------------------------- | --------------------------------------------------------------- |
| `-s`, `--separator <text>` | Set the output delimiter. The default is a tab.                 |
| `-f`, `--format <format>`  | Set the output format to `csv` or `json`. The default is `csv`. |
| `-d`, `--days <days>`      | Aggregate from this many days ago through today.                |
| `--clipboard`              | Copy the output to the clipboard as well as stdout.             |
| `-h`, `--help`             | Show command-line help.                                         |
| `--no-project`             | Omit the project column from CSV output.                        |
| `--no-date`                | Omit the date header row from CSV output.                       |
| `--version`                | Show the CLI version.                                           |

### Update the CLI

Update the current compiled executable in place:

```sh
toggl update
toggl update --channel nightly
toggl update --channel stable
```

Versions named `nightly-YYYYMMDD-<sha>` use the Nightly channel by default; all
other versions use the latest stable GitHub release. An explicit `--channel`
overrides that selection. Self-update supports Linux x64 and macOS arm64,
matching the published `.tar.gz` artifacts. Windows and other architectures must
be updated manually.

The updater verifies the downloaded SHA-256 checksum and binary version before
atomically replacing the running executable. It therefore needs `tar` on `PATH`
and write permission for the executable's directory. A failure leaves the
existing binary unchanged. Self-update is unavailable under `deno task run` (or
another source execution); install a compiled release binary first.

### Aggregate time entries

Specify the inclusive start and end dates in `YYYY-MM-DD` format. Date ranges
may cross month and year boundaries.

```sh
deno task run -- summary 2026-06-01 2026-06-15
```

Alternatively, use `--days` or `-d` to aggregate from the specified number of
days ago through today. Today is determined using the configured `timezone`, or
the execution environment's timezone when no timezone is configured. Both
endpoints are included, so `--days 7` outputs eight days including today.

```sh
deno task run -- summary --days 7
deno task run -- summary -d 7
```

By default, the command outputs a single tab-separated table with projects in
the first column and work time in minutes for each project and date in the
remaining columns. You can paste this output directly into spreadsheet
applications such as Excel.

Use `--separator` or `-s` to change the delimiter:

```sh
deno task run -- summary --separator "," 2026-06-01 2026-06-15
```

Use `--no-project` to omit the project column from CSV output:

```sh
deno task run -- summary --no-project 2026-06-01 2026-06-15
```

Use `--no-date` to omit the date header row from CSV output:

```sh
deno task run -- summary --no-date 2026-06-01 2026-06-15
```

The two options can be combined to output only the work-time values:

```sh
deno task run -- summary --no-project --no-date 2026-06-01 2026-06-15
```

Use `--format json` or `-f json` to output JSON:

```sh
deno task run -- summary --format json 2026-06-01 2026-06-15
```

Use `--clipboard` to print the summary and copy the same output to the
clipboard:

```sh
deno task run -- summary --clipboard 2026-06-01 2026-06-15
```

The JSON output maps each date to project IDs and their work time in minutes:

```json
{
  "2026-06-01": {
    "123456789": 60
  }
}
```

### List projects

List the display names of all active, visible projects:

```sh
deno task run -- projects
```

Project information can also be output as JSON:

```sh
deno task run -- --format json projects
```

Print the CLI version:

```sh
deno task run -- --version
```

To add all active Toggl projects that are not yet in the configuration file,
run:

```sh
deno task run -- projects sync
```

Each new project is appended with its Toggl project name as a comment and with
`hidden = false`. Existing project settings and other configuration file content
are left unchanged.

### Show configuration

Show the loaded configuration values:

```sh
deno task run -- config
```

Configuration can also be output as JSON:

```sh
deno task run -- config --format json
```

The `TOKEN` setting is never printed.

## Build

Compile a standalone binary:

```sh
deno task compile
./toggl --version
```

The `build` task is a convenience alias that writes the binary to `out/toggl`:

```sh
deno task build
```

Run the compiled executable as follows:

```sh
./out/toggl summary 2026-06-01 2026-06-15
./out/toggl summary --no-project 2026-06-01 2026-06-15
./out/toggl summary --clipboard 2026-06-01 2026-06-15
./out/toggl projects
./out/toggl projects sync
./out/toggl config
```

If `--version` is omitted, the compiled binary reports the development version
`0.0.0-dev`. Pass the release version explicitly when building release binaries:

```sh
deno task compile --version 0.1.0
```

## Release Archives

Build release archives under `dist/`:

```sh
deno task dist --version 0.1.0
```

The release build creates archives for:

- `darwin-arm64`
- `linux-x64`
- `windows-x64`

To build a single target:

```sh
deno task dist --version 0.1.0 --target linux-x64
```

Each archive includes the `toggl` binary and `README.md`. If a `LICENSE` file is
present, it is included as well. macOS and Linux targets are packaged as
`.tar.gz`; Windows is packaged as `.zip`. `dist/checksums.txt` and per-archive
`.sha256` files are generated for the final archives.

The release build uses the system `tar` command for `.tar.gz` archives and the
system `zip` command for the Windows archive. A full release build requires both
commands. A single-target build only requires the archive command for that
target.

For native targets, the release build runs the compiled binary with `--version`
to verify that the requested release version was embedded.

Before creating a stable release, verify the local checkout and the successful
GitHub `Test` workflow for the commit on `main`:

```sh
deno task release:check --version 1.0.0
```

The check requires a clean `main` branch that exactly matches `origin/main`, an
unused version tag and GitHub release, and a successful workflow run for the
target commit. For the first stable release, create the `release-notes-baseline`
tag on the repository root commit. Later releases use the latest non-draft,
non-prerelease stable release and ignore Nightly tags.

Preview GitHub's generated title and notes as reviewable JSON on standard
output, or write them to a file:

```sh
deno task release:notes --version 1.0.0
deno task release:notes --version 1.0.0 --output /tmp/release-notes.json
```

Pushing a semantic-version tag, with or without a leading `v` (for example,
`v0.1.0` or `0.1.0`), publishes all three archives and their checksum files as a
GitHub release. Release notes start at the previous stable release; the
`release-notes-baseline` tag is used when there is no previous stable release.

After the `Test` workflow succeeds on `main`, the tested commit is published as
the `nightly` prerelease. Its moving `nightly` tag and assets are replaced on
each successful run independently of stable releases. Nightly binaries report
their version as `nightly`, and archives use names such as
`toggl-cli-nightly-linux-x64.tar.gz`. Legacy versioned Nightly releases and tags
are removed automatically.

## Running the Installed Command

Run the installed command as follows:

```sh
toggl summary 2026-06-01 2026-06-15
toggl summary --clipboard 2026-06-01 2026-06-15
toggl projects
toggl config
```

## Development

When running from a checkout without installing the executable, use
`deno task run --` and pass the same arguments after it:

```sh
deno task run -- init
deno task run -- summary 2026-06-01 2026-06-15
deno task run -- projects
```

```sh
deno fmt --check
sh -n install.sh
deno check --lock=deno.lock main.ts main_test.ts scripts/install.ts scripts/install_release_test.ts scripts/install_test.ts scripts/release.ts scripts/release_test.ts toggl/date_range_test.ts
deno test --allow-read --allow-write --allow-run=sh,tar --allow-env=HOME,PATH main_test.ts scripts/install_release_test.ts scripts/install_test.ts scripts/release_test.ts toggl/date_range_test.ts
deno compile --quiet --allow-net --allow-read --allow-write --allow-run=pbcopy,wl-copy,xclip,xsel,clip,powershell.exe,powershell --allow-env --output /tmp/toggl-cli main.ts
```

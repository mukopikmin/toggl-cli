# toggl-cli

A Deno CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values or JSON.

## Requirements

- Deno 2.8 or later
- A Toggl Track API token
- The ID of the target workspace

## Configuration

Create a config file:

```sh
deno task run -- init
```

This creates `~/.config/toggl-cli/config.toml` if it does not already exist. You
can also create it manually:

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

[projects."789012"]
hidden = true
```

Display names are used when rendering project lists and summary CSV output. When
`display_name` is omitted, the Toggl project name is used. When `hidden` is
omitted, it defaults to `false`. Hidden projects are excluded from `projects`
output and summary CSV output.

The optional `timezone` setting is used to calculate the Toggl time entry query
range. When it is omitted, the CLI preserves the existing UTC-based behavior.

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

### Aggregate time entries

Specify the start and end days as day numbers in the current month. The end day
is included in the aggregation.

```sh
deno task run -- summary 1 15
```

The legacy root form remains supported:

```sh
deno task run -- 1 15
```

By default, the command outputs a list of visible projects followed by work time
in minutes for each project and date. Columns are separated by tabs.

Use `--lastMonth` or `-l` to aggregate the previous month:

```sh
deno task run -- --lastMonth summary 1 31
```

Use `--separator` or `-s` to change the delimiter:

```sh
deno task run -- --separator "," summary 1 15
```

Use `--format json` or `-f json` to output JSON:

```sh
deno task run -- --format json summary 1 15
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
./out/toggl summary 1 15
./out/toggl --lastMonth summary 1 31
./out/toggl projects
./out/toggl projects sync
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

## Install

On macOS, build in a temporary directory and install the executable to
`$HOME/.local/bin/toggl`:

```sh
deno task install:mac
```

Make sure `$HOME/.local/bin` is included in your `PATH`, then run the installed
command as follows:

```sh
toggl 1 15
toggl projects
```

## Development

```sh
deno fmt --check
deno check --lock=deno.lock main.ts main_test.ts scripts/compile.ts scripts/build_release.ts scripts/install_macos.ts toggl/date_range_test.ts
deno test
>>>>>>> refs/remotes/origin/main
```

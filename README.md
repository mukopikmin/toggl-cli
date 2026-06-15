# toggl-cli

A Deno CLI that aggregates Toggl Track time entries by project and date. Results
can be output as delimiter-separated values or JSON.

## Requirements

- Deno 2.8 or later
- A Toggl Track API token
- The ID of the target workspace

## Configuration

Create `~/.toggl_config` in your home directory:

```ini
WORKSPACE=your_workspace_id
TOKEN=your_api_token
TIMEZONE=Asia/Tokyo
```

`WORKSPACE` and `TOKEN` are required. `TIMEZONE` is optional, but specifying an
IANA time zone name is recommended so daily totals align with your local time
zone.

You can find your API token in your Toggl Track profile settings. Because the
configuration file contains credentials, restrict its permissions so that other
users cannot read it:

```sh
chmod 600 ~/.toggl_config
```

## Usage

### Aggregate time entries

Specify the start and end days as day numbers in the current month. The end day
is included in the aggregation.

```sh
deno task run -- 1 15
```

By default, the command outputs a list of active projects followed by work time
in minutes for each project and date. Columns are separated by tabs.

Use `--lastMonth` or `-l` to aggregate the previous month:

```sh
deno task run -- --lastMonth 1 31
```

Use `--separator` or `-s` to change the delimiter:

```sh
deno task run -- --separator "," 1 15
```

Use `--format json` or `-f json` to output JSON:

```sh
deno task run -- --format json 1 15
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

List the names of all active projects:

```sh
deno task run -- projects
```

Project information can also be output as JSON:

```sh
deno task run -- --format json projects
```

## Build

Build a standalone executable at `out/toggl`:

```sh
deno task build
```

Run the compiled executable as follows:

```sh
./out/toggl 1 15
./out/toggl --lastMonth 1 31
./out/toggl projects
```

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
deno check --lock=deno.lock main.ts main_test.ts scripts/install_macos.ts toggl/date_range_test.ts
deno test
```

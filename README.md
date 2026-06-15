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
deno task dev init
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
deno task dev -- 1 15
```

By default, the command outputs a list of visible projects followed by work time
in minutes for each project and date. Columns are separated by tabs.

Use `--lastMonth` or `-l` to aggregate the previous month:

```sh
deno task dev -- --lastMonth 1 31
```

Use `--separator` or `-s` to change the delimiter:

```sh
deno task dev -- --separator "," 1 15
```

Use `--format json` or `-f json` to output JSON:

```sh
deno task dev -- --format json 1 15
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
deno task dev -- projects
```

Project information can also be output as JSON:

```sh
deno task dev -- --format json projects
```

To add all active Toggl projects that are not yet in the configuration file,
run:

```sh
deno task dev -- projects sync
```

Each new project is appended with its Toggl project name as a comment and with
`hidden = false`. Existing project settings and other configuration file content
are left unchanged.

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
./out/toggl projects sync
```

## Development

```sh
deno fmt --check
deno check --lock=deno.lock main.ts main_test.ts toggl/date_range_test.ts
deno test
```

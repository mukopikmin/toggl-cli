# toggl-cli

CLI for listing Toggl projects and summarizing time entries.

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

## Usage

List projects:

```sh
deno task dev projects
deno task dev projects --format json
```

Summarize time entries for the current month:

```sh
deno task dev summary 1 31
deno task dev summary --format json 1 31
```

Use `--lastMonth` to target the previous month:

```sh
deno task dev summary --lastMonth 1 31
```

Summary CSV output uses tabs by default. Use `--separator` to change the
separator:

```sh
deno task dev summary --separator "," 1 31
```

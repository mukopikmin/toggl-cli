# toggl-cli

CLI for listing Toggl projects and summarizing time entries.

## Configuration

Create a config file:

```sh
deno task dev init
```

Create `~/.config/toggl-cli/config.toml`:

```toml
workspace = "your_workspace_id"
token = "your_api_token"
```

Optional per-project settings can be configured with the `projects` table:

```toml
[projects."123456"]
display_name = "Client A"
hidden = false

[projects."789012"]
hidden = true
```

Display names are used when rendering project lists and summary CSV output.
Hidden projects are excluded from project lists and summary CSV output.

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
deno task dev 1 31
deno task dev --format json 1 31
```

Use `--lastMonth` to target the previous month:

```sh
deno task dev --lastMonth 1 31
```

CSV output uses tabs by default. Use `--separator` to change the separator:

```sh
deno task dev --separator "," 1 31
```

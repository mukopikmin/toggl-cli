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

Print the CLI version:

```sh
deno task dev --version
```

## Development

Run tests:

```sh
deno test --allow-net --allow-read --allow-write --allow-env
```

Compile a standalone binary:

```sh
deno task compile
./toggl --version
```

The existing `build` task is kept as a convenience alias that writes the binary
to `out/toggl`:

```sh
deno task build
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

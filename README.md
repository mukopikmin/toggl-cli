# toggl-cli

A command-line tool for summarizing Toggl Track time by project and date.

## Install

On Linux x64 or macOS arm64, run:

```sh
curl -fsSL https://raw.githubusercontent.com/mukopikmin/toggl-cli/main/install.sh | sh
```

Add `$HOME/.local/bin` to your `PATH` if needed. On Windows x64, download the
archive from [GitHub Releases](https://github.com/mukopikmin/toggl-cli/releases).

The installer also sets up completion for Bash, Zsh, and Fish. Run
`toggl completion <shell>` to print a completion script.

## Quick start

Run `toggl init` to enter your Toggl Track workspace ID, API token, and time
zone. This creates `~/.config/toggl-cli/config.toml`.

To view a summary for the last seven days, including today, run:

```sh
toggl summary --days 7 --format table
```

Run `toggl --help` for commands and options, or `toggl update` to update a
release binary.

## Architecture

- `cmd/toggl` assembles dependencies and dispatches commands.
- `internal/command` handles CLI arguments, terminal interaction, and output.
- `internal/usecase` coordinates time-entry retrieval, project selection, and
  project sync.
- `internal/model` holds settings, time entries, and pure calculation rules.
- `internal/config` reads and writes `config.toml` and `templates.toml`.
- `internal/toggl` sends API requests and converts responses to models.
- `internal/selfupdate` verifies and installs release updates.

## Development

Install Make and the Go version specified in [go.mod](go.mod). Run `make build`
to build `out/toggl` and `make check` to run the standard checks.
Run `make test-e2e` for end-to-end tests with Docker.

[Third-party notices](THIRD_PARTY_NOTICES.md)

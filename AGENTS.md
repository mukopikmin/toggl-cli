# Repository Guidelines

## Changes

- Keep changes scoped to the task. Follow existing Go conventions and prefer the
  standard library. Add dependencies or abstractions only when justified.
- Keep implementation, tests, and documentation in sync. Update the README and
  related docs when their commands, options, configuration, outputs, Go
  requirements, or referenced paths change. Verify examples against the current
  implementation, `go.mod`, and `Makefile` as applicable.
- Never commit credentials, API tokens, or local configuration files.

## CLI and package boundaries

- Use singular resource names with explicit verb subcommands, such as
  `project list` and `project sync`. Require a subcommand for resources. Keep
  standalone operations such as `summary` and `init` at the top level.
- Use the same command names in parsing, dispatch, help, tests, and docs.
- Keep `cmd/toggl/main.go` focused on argument dispatch and dependency assembly.
  Use `internal/command` for CLI input, coordination, and output formatting.
- Keep API calls, private DTOs, and conversion in `internal/toggl`. Its public
  methods return domain models. Keep `internal/model` independent of the API
  adapter, with pure date calculations and aggregation where possible.
- Keep configuration loading and validation in `internal/config`. Leave
  command-specific behavior in `internal/command`.

## Tests and CI

- Test behavior changes on normal paths, boundaries, and major error cases. Add
  a regression test for bug fixes when practical. For date and timezone changes,
  cover both ends of a month and dates that differ from UTC.
- Test API-dependent code through the client boundary using test doubles or
  local test servers. Keep real API calls out of unit tests.
- Before every push, inspect the current workflows in `.github/workflows/` and
  run the applicable checks against the exact changes being pushed. Use
  `.github/workflows/test.yml` as the source for CI checks. Rerun affected checks
  after further edits. Fix failures without weakening checks, and explain any
  necessary changes to the checks themselves.
- Verify `go.mod` and `go.sum` after changing dependencies. Review `git diff`
  and `git status` for unintended changes before reporting results and any
  verification blockers.

## GitHub

- Use concise English for branch names, commits, issues, pull requests, and
  reviews. Use descriptive pull request titles without tool prefixes. Add
  `Closes #<issue_number>` to the pull request when resolving an issue.
- If `gh auth status` fails in the sandbox, retry with network access before
  diagnosing credentials. Use GitHub MCP if authenticated `gh` is unavailable.
  Ask the user to reauthenticate only if the network-enabled check confirms an
  invalid or expired token and no authenticated GitHub tool is available.
  Ask for help with GitHub operations only if neither tool is available. Diagnose
  Git remote authentication separately.

# Repository Guidelines

## General Principles

- Keep changes limited to the requested objective. Do not include unrelated
  refactoring.
- Prefer the existing design, naming conventions, and Deno standard
  functionality. Add dependencies or abstractions only when their necessity can
  be clearly justified.
- Treat implementation, tests, and documentation as a single unit. Do not leave
  any of them outdated.
- Never commit credentials, API tokens, or local configuration files.

## Language

- Use English for commit messages, branch names, pull request titles and
  descriptions, issue text, and review comments.
- Use clear, concise English that explains the purpose and impact of a change.
  Avoid mixing languages in repository and GitHub communication.
- Do not add tool-specific prefixes such as `[codex]` to pull request titles.
  Write titles that describe the change directly.

## Tool Usage

- If `gh auth status` fails inside the sandbox, do not immediately conclude that
  the token is invalid. Run `gh auth status` again with network-enabled or
  escalated execution.
- Ask the user to reauthenticate only when the network-enabled or escalated
  check also confirms that the token is invalid or expired.
- If an authenticated GitHub CLI is unavailable but GitHub MCP is available, use
  GitHub MCP for repository and pull request operations.
- Treat Git remote authentication, including authentication used by `git push`,
  independently from GitHub CLI authentication.
- Ask the user to authenticate or intervene only when neither an authenticated
  GitHub CLI nor GitHub MCP is available.

## Documentation Consistency

- When changing commands, options, configuration fields, required Deno versions,
  output formats, or module structure, update `README.md` and any related
  documentation in the same change.
- Ensure code examples and commands work with the current implementation. Remove
  references to obsolete APIs, deleted options, and nonexistent files.
- When CI, `deno.json`, and the README describe the same procedure, verify that
  they agree. Treat `.github/workflows/test.yml` as the source of truth for the
  checks performed by CI.
- Even for documentation-only changes, verify the content against the code and
  configuration files.

## Separation of Responsibilities

- Keep `main.ts` focused on argument parsing, dependency assembly, and
  delegation to commands. Do not add API communication or complex aggregation
  logic directly to it.
- Use `command/` for CLI use cases, input interpretation, and output formatting.
- Use `toggl/` for Toggl API communication, API data types, and domain logic
  such as date ranges.
- Keep `config.ts` responsible for configuration loading and validation. Do not
  add command-specific behavior to it.
- Split a file or function when it begins to hold multiple independent
  responsibilities. Do not introduce an abstraction that has only one caller and
  does not reduce complexity.
- Separate external communication, date calculations, aggregation, and output
  formatting. Prefer pure functions that can be tested without side effects.

## Testing

- When behavior changes, add or update tests covering boundary values and major
  error conditions as well as the normal path.
- For changes involving dates, time zones, or month boundaries, explicitly test
  the beginning and end of a month and date differences from UTC.
- Test code that depends on external APIs through the client boundary. Regular
  unit tests must not call the real API.
- For bug fixes, add a regression test that fails before the fix whenever
  practical.

## Required Checks Before Push

- Immediately before every push, inspect the latest workflows under
  `.github/workflows/` and run all applicable CI checks locally.
- Do not rely on a previously remembered command list, an earlier run, or the
  commands documented in the README. The current CI workflows are the source of
  truth.
- Fix the cause of a failed check. Do not make a check pass by disabling,
  excluding, or weakening it. If a check itself must change, clearly explain the
  reason and impact.
- After changing dependencies, including `deno.lock`, verify that the lockfile
  and import definitions remain consistent.
- If code changes after the final checks, rerun the affected checks. The
  verification results must correspond to the exact contents being pushed.

## Completing Changes

- Review `git diff` and `git status` for unintended files, generated artifacts,
  or credentials.
- Briefly report the changes made, documentation updated, and checks run.
- If any check could not be run, state the reason and the remaining risk.

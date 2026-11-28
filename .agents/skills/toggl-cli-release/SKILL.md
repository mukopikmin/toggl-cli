---
name: toggl-cli-release
description: Use when asked to prepare, publish, or verify a Toggl CLI stable release, with or without a version. Propose a semantic version, obtain approval for the exact release, then push the stable tag and monitor publication.
---

# Toggl CLI Release

Communicate in the user's language. Use the `Release` workflow in
[release.yml](../../../.github/workflows/release.yml) to validate, build, attest,
and publish the release. Respect its validation gates and use its artifacts for
publication.

## Inspect and propose

1. Read the repository instructions and inspect status, worktrees, branches,
   remote tags, releases, and open pull requests. Preserve local changes and the
   current branch. The local checkout may differ from the release target.
2. Run `git fetch --no-tags origin main` and resolve the full SHA of `origin/main`.
   Confirm this commit contains the tag-triggered workflow. Require a successful
   `Test` push run on `main` for that exact SHA before release.
3. Find the newest stable release, excluding drafts, prereleases, `nightly`, and
   `v0.0.0-nightly`. Fetch only the fixed comparison tag if needed. If there is
   no stable release but stable version tags exist, investigate before proceeding.
4. For a first stable release, use `release-notes-baseline`. Verify it points to
   the root commit. If it is missing, show the root SHA and get approval before
   creating and pushing the lightweight baseline tag. Stop if it points elsewhere.
5. Review commits and merged pull requests from the comparison tag through the
   target SHA. Stop if the changes do not justify a release. Propose an unused
   semantic version newer than the last stable release, without a `v` prefix or
   leading zeroes. For a user-specified version, explain any mismatch with the
   changes. Use a major bump for breaking changes at `1.x` or later, a minor bump
   for compatible features or breaking changes at `0.x`, and a patch bump for
   fixes, documentation, or maintenance.

## Approve and push

Present the current and proposed versions, bump rationale, full target SHA,
comparison tag, changes and merged pull requests, successful `Test` run URL,
and open pull requests omitted from the release. Explain that pushing the tag
starts automatic publication using that notes range. Obtain explicit approval
of the version, SHA, notes range, and omissions before creating or pushing the
tag. Reuse approval while those inputs stay the same.

Immediately before pushing, fetch `origin/main` with `--no-tags` again. Verify
that the approved SHA is still its tip, the previous stable release and notes
range match, and the proposed tag and release are still unused. Return to
inspection and approval if any reviewed input changes.

Create the annotated tag at the approved SHA, with exactly one
`Previous-Stable-Tag` trailer containing the approved comparison tag. Substitute
the approved values in these commands:

```sh
git tag --annotate 'v<version>' '<approved-full-sha>' \
  --message 'Release v<version>' \
  --message 'Previous-Stable-Tag: <approved-comparison-tag>' &&
  git push origin 'refs/tags/v<version>'
```

Use authorized user or GitHub App credentials. A push using the repository
`GITHUB_TOKEN` will not start the workflow. Keep credentials private. Push only
the approved tag and let the workflow publish the GitHub release. Never force,
move, delete, or reuse a stable tag.

## Monitor and verify

1. Match the `Release` run to the tag push using the event, ref, SHA, and creation
   time. Monitor it through completion.
2. On success, confirm the release is stable and latest, uses the approved SHA
   and notes range, and includes the expected merged pull requests while omitting
   open ones. Check that it has three platform archives, their individual SHA-256
   files, and `checksums.txt`. Use the workflow results for checksum and provenance
   verification.
3. Report the release URL, tag, full SHA, notes range, assets, verification
   results, and omitted pull requests.

## Failures and retries

- If the tag exists without a release or its run fails, inspect that exact run
  and report the run URL, failing step, relevant logs, and any partial state.
  Leave the tag and any partial release intact.
- Get approval before rerunning a failed run. Rerun only with the unchanged tag
  and target. A fix on a new commit needs a new unused version and approval.
- If the release already exists, verify and report it. Get new approval before
  repairing a failed published release, and never rerun publication over it.

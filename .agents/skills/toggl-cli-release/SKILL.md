---
name: toggl-cli-release
description: Review changes, propose a semantic version, obtain approval for the exact main commit, then push a stable tag and monitor the automatic GitHub Actions release. Use when asked to prepare, cut, publish, or verify a Toggl CLI stable release, whether or not a version is specified.
---

# Toggl CLI Release

The skill handles release judgment, user approval, tag operations, and
reporting. The `Release` workflow in `.github/workflows/release.yml` enforces
every required machine check, builds and attests artifacts, generates notes, and
publishes the GitHub release. Communicate in the user's language.

## Safety Rules

- Release only the exact tested tip of `main`. Local inspection is advisory;
  GitHub Actions validates it again before building and publishing.
- Never delete, stash, commit, or overwrite user changes to obtain a clean tree.
  Do not switch branches or require local `HEAD` to match the release target.
- Never force, move, delete, or reuse a stable release tag.
- A stable tag push starts automatic publication. Obtain explicit approval for
  the version, full target SHA, notes range, and omitted pull requests before
  creating or pushing the tag. Approval of a legacy preparation run does not
  authorize a tag push under this workflow.
- Invalidate approval if the target, current `origin/main`, previous stable
  release, or reviewed change range changes before the push.
- Never bypass failed validation or publish locally built artifacts.
- Ignore `nightly`, `v0.0.0-nightly`, drafts, and prereleases when selecting the
  stable notes baseline.
- Do not repair or overwrite a failed published release without new approval.

## Workflow

### 1. Inspect and Propose

1. Read repository instructions. Inspect status, worktrees, branches, remote
   tags, releases, open pull requests, and release workflows without changing
   the working tree. Confirm the tag-triggered workflow is already on
   `origin/main`; workflow changes must be merged and tested before releasing.
2. Fetch `origin/main` using `git fetch --no-tags origin main` and resolve its
   full SHA. Do not use `git fetch --tags`: the rolling nightly tag moves.
3. Identify the newest non-draft, non-prerelease stable release. For the first
   stable release, use `release-notes-baseline`. Fetch only the fixed comparison
   tag if it is missing locally.
4. Review commits and merged pull requests from that tag through the exact
   `origin/main` SHA. Stop if there are no changes that justify a release.
5. Propose one unprefixed semantic version without leading zeroes, newer than
   the previous stable release, with no existing tag or release. If the user
   supplied a version, compare it with the changes and explain any suspected
   semantic-version mismatch. Choose the increment as follows:
   - increment major for breaking changes when the current major is at least 1;
   - increment minor for backward-compatible features, and for breaking changes
     while the project is on `0.x`;
   - increment patch for fixes, documentation, or maintenance only.
6. Inspect the successful `Test` push run on `main` for that exact SHA. A
   pending or failed run is not sufficient; do not push a release tag while it
   is pending or failed. Actions remains the authoritative validator.
7. For the first stable release only, verify `release-notes-baseline` points to
   the repository root commit. If absent, show the root SHA and ask for approval
   before creating and pushing that lightweight non-release tag. If it points
   elsewhere, stop.

### 2. Approve Publication

Present the current stable version, proposed version and increment rationale,
full target SHA, comparison tag, principal changes and merged pull requests,
successful Test run URL, and all open pull requests the release will omit.

Explain that approving the tag push authorizes automatic build and publication:
release notes will be generated in Actions from the displayed range, with no
later notes/digest approval gate. Ask the user to approve that concrete release
or provide another version. This is a mandatory gate before tag creation/push.
Existing explicit approval of these details need not be requested again unless
one of the reviewed inputs changes.

### 3. Push the Approved Tag

1. Immediately before pushing, fetch `origin/main` with `--no-tags` and verify
   its full SHA, the previous stable release, and tag/release absence still
   match the proposal. If anything changed, return to inspection and approval.
2. Create an annotated tag on the approved SHA, without checking out that
   commit:

   ```sh
   git tag --annotate v<version> <approved-full-sha> --message "Release v<version>"
   git push origin refs/tags/v<version>
   ```

3. Use the user's authorized Git credentials. A tag pushed with the repository's
   `GITHUB_TOKEN` does not trigger another workflow; use an authorized user or
   GitHub App credential instead. Never print credentials.
4. Push only that tag. Do not dispatch a legacy Prepare/Publish workflow, create
   a GitHub release directly, or download candidate artifacts locally.

### 4. Monitor and Verify

1. Find the exact `Release` run for the tag push and approved SHA, using its
   event, tag/head branch, head SHA, and creation time. Monitor until
   completion.
2. On failure, report the run URL, failing step, and relevant logs. Leave the
   tag and any partial release intact. Do not move the tag or overwrite assets.
3. On success, inspect the workflow verification and release metadata. Confirm
   it is stable and latest, uses the approved target and notes range, contains
   the expected merged pull requests, and omits open pull requests.
4. Confirm all three platform archives, their individual SHA-256 files, and
   `checksums.txt` are attached. Actions verifies checksums and provenance for
   this workflow and the tagged source commit before publication. Local artifact
   downloads or local attestation verification are not required.
5. Report the release URL, tag, full SHA, notes range, assets, verification
   results, and omitted open pull requests.

## Retry Rules

- If a tag exists but no release exists, inspect its exact run and report the
  partial state. Never delete/recreate the tag to retrigger publication.
- After explaining a failed run, obtain approval before rerunning it. Reuse only
  an unchanged tag and target; Actions must still pass every check. A fix on a
  new commit requires a new unused version/tag and a new approval.
- If a release already exists, verify and report it instead of publishing a
  duplicate. Do not rerun publication over an existing release.

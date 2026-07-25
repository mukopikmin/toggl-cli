---
name: toggl-cli-release
description: Propose a semantic version, then prepare, publish, monitor, and verify Toggl CLI stable releases with explicit approval gates, exact-SHA checks, generated release-note review, and artifact verification. Use when asked to prepare, cut, publish, or verify a Toggl CLI stable release, whether or not a version is specified.
---

# Toggl CLI Release

Treat a stable release as a guarded process. Communicate in the user's language
and follow `AGENTS.md`. Use `.github/workflows/test.yml` as the source of truth
for checks and `.github/workflows/release.yml` as the source of truth for
publication behavior.

## Safety Rules

- Release only the exact tested commit on `main` that equals `origin/main`.
- Never delete, stash, commit, or overwrite user changes to clean the tree.
- Never force, move, or reuse a stable release tag.
- Never push a release tag before the user approves the notes and exact SHA.
- Invalidate approval when the SHA, notes start tag, title, or body changes.
- Never release from a failed or missing `Test` workflow run.
- Exclude `nightly` and all prereleases from the stable notes baseline.
- Do not repair or replace a failed published release without new approval.

## Workflow

### 1. Inspect and Propose

1. Read repository instructions. Without modifying anything, inspect status,
   worktrees, local and remote branches, stable tags and releases, open pull
   requests, and both test and release workflows.
2. Find the latest non-draft, non-prerelease release. Use its tag as the notes
   baseline. For the first stable release, use `release-notes-baseline`.
3. Review commits and merged pull requests between the baseline and
   `origin/main`. Propose one unprefixed semantic version:
   - increment major for breaking changes after `1.0.0`;
   - increment minor for backward-compatible features and for breaking changes
     during `0.x`;
   - increment patch for fixes, documentation, or maintenance only.
4. Choose the next unused version for that increment. Stop if the changes do not
   justify a release.
5. Present the current and proposed versions, increment rationale, important
   changes, and open pull requests that would be omitted. Ask for approval. Do
   not synchronize, build, or tag before approval.
6. If the user supplied a version, still check it against the changes. Require
   explicit confirmation for a suspected mismatch.
7. Require an unprefixed `X.Y.Z` version and derive the tag as `vX.Y.Z`.

### 2. Synchronize `main`

1. Stop if tracked or untracked files would be overwritten.
2. Run `git fetch origin --prune` without `--tags`; the moving `nightly` tag can
   differ locally and remotely.
3. Switch to `main` and fast-forward only to `origin/main`. Never merge a
   divergent branch.
4. For the first release, verify that `release-notes-baseline` points to the
   root commit. If absent, show the root SHA and ask before creating and pushing
   that lightweight, non-release tag. Stop if it points elsewhere.
5. Fetch only the stable baseline tag needed to generate notes when it is not
   available locally. Do not fetch `nightly` with a forced refspec.

### 3. Run Preflight

1. Confirm the branch is `main`, the tree is clean, and `HEAD` equals
   `origin/main`.
2. Confirm that neither the proposed stable tag nor its GitHub release exists.
3. Verify a successful `Test` workflow exists for the exact full `HEAD` SHA,
   using `gh run list` and `gh run view`; do not rely only on branch status.
4. Re-read `.github/workflows/test.yml` and run every applicable CI command
   exactly as declared there.
5. Build all release artifacts in a clean `dist/` directory:

   ```sh
   rm -rf dist
   deno task dist --version <version>
   ```

6. Verify that `dist/` contains the three platform archives, one `.sha256` file
   per archive, and `checksums.txt`. Verify every recorded SHA-256 digest.

### 4. Generate and Review Notes

Preview the notes without creating a release:

```sh
gh api --method POST "repos/{owner}/{repo}/releases/generate-notes" \
  -f tag_name="v<version>" \
  -f target_commitish="<full-sha>" \
  -f previous_tag_name="<stable-baseline>"
```

Present the exact target SHA, baseline, title, and generated body. Verify that
the changelog is non-empty, expected merged pull requests are present, open pull
requests are absent, and the text contains nothing private or unsuitable. Ask
the user to approve both the notes and SHA. This is a mandatory blocking gate.

### 5. Revalidate and Tag

After approval:

1. Fetch again without changing the approved commit.
2. Repeat all preflight checks and regenerate the notes.
3. Compare the SHA, baseline, title, and body with the approved values. Discard
   approval and return to review if any value changed.
4. Create an annotated `v<version>` tag on the full approved SHA and inspect it.
5. Push only that tag, without force. The tag push starts the `Release`
   workflow; do not create the GitHub release manually.

### 6. Monitor and Verify

1. Monitor the `Release` workflow for the tagged SHA until completion.
2. On failure, report the failing step and relevant logs. Do not move the tag or
   replace the release automatically.
3. On success, verify that the GitHub release is stable and latest, points to
   the approved SHA, and uses the approved notes range.
4. Verify that all three platform archives, their three `.sha256` files, and
   `checksums.txt` are attached. Download and validate checksums when practical.
5. Report the release URL, tag, SHA, notes baseline, assets, verification
   results, and omitted open pull requests.

## Retry Rules

- Before pushing the stable tag, restart safely from inspection or preflight.
- Reuse the first-release baseline only when it points to the root commit.
- If the stable tag exists at the approved SHA but no release exists, stop and
  ask whether to retry the workflow; do not recreate or push the tag.
- If a release already exists, verify and report it instead of duplicating it.

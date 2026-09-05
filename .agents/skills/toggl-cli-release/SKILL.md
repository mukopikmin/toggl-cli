---
name: toggl-cli-release
description: Propose a semantic version, then prepare, publish, monitor, and verify Toggl CLI stable releases with explicit approval gates, exact-SHA checks, generated release-note review, and artifact verification. Use when asked to prepare, cut, publish, or verify a Toggl CLI stable release, whether or not a version is specified.
---

# Toggl CLI Release

Treat a stable release as a guarded process. Communicate in the user's language
and follow `AGENTS.md`. Use `.github/workflows/test.yml` as the source of truth
for checks, `.github/workflows/release_prepare.yml` for candidate preparation,
and `.github/workflows/release.yml` for publication.

## Safety Rules

- Release only the exact tested commit on `main` that equals `origin/main`.
- Never delete, stash, commit, or overwrite user changes to clean the tree.
- Never force, move, or reuse a stable release tag.
- Never dispatch publication before the user approves the exact SHA, release
  notes, and release-plan SHA-256.
- Invalidate approval when the preparation run, SHA, baseline, title, body, plan
  hash, or artifact set changes.
- Never release from a failed or missing `Test` workflow run.
- Exclude `nightly` and all prereleases from the stable notes baseline.
- Do not repair or replace a failed published release without new approval.

## Workflow

### 1. Inspect and Propose

1. Read repository instructions. Without modifying anything, inspect status,
   worktrees, local and remote branches, stable tags and releases, open pull
   requests, and the test, preparation, and publication workflows.
2. Find the latest non-draft, non-prerelease release. Use its tag as the notes
   baseline. For the first stable release, use `release-notes-baseline`.
3. Review commits and merged pull requests between the baseline and
   `origin/main`. Propose one unprefixed semantic version: increment major for
   breaking changes after `1.0.0`, minor for features or breaking changes during
   `0.x`, and patch for fixes, documentation, or maintenance only.
4. Choose the next unused version for that increment. Stop if the changes do not
   justify a release. If the user supplied a version, check it against the
   changes and require confirmation for a suspected mismatch.
5. Present the current and proposed versions, rationale, important changes, and
   open pull requests that would be omitted. Ask for approval before proceeding.
6. Require an unprefixed `X.Y.Z` version and derive the tag as `vX.Y.Z`.

### 2. Synchronize and Preflight `main`

1. Stop if tracked or untracked files would be overwritten.
2. Run `git fetch origin --prune` without `--tags`; the moving `nightly` tag can
   differ locally and remotely.
3. Switch to `main` and fast-forward only to `origin/main`. Never merge a
   divergent branch.
4. For the first release, verify that `release-notes-baseline` points to the
   root commit. If absent, show the root SHA and ask before creating and pushing
   that lightweight, non-release tag. Stop if it points elsewhere.
5. Confirm the tree is clean and `HEAD` equals `origin/main`; confirm the stable
   tag and release are unused; and verify a successful `Test` run exists for the
   exact full SHA with `gh run list` and `gh run view`.
6. Re-read `.github/workflows/test.yml` and run every applicable CI command
   exactly as declared there.

### 3. Prepare and Review the Candidate

1. Dispatch preparation with the unprefixed version and full SHA:

   ```sh
   gh workflow run release_prepare.yml \
     -f version=<version> \
     -f target_sha=<full-sha>
   ```

2. Locate the dispatched run, record its run ID, and monitor it until it
   succeeds. Download the `release-candidate` artifact from that exact run.
3. Verify its checksums. Calculate the SHA-256 of `release-plan.json` and
   compare it with the preparation run summary.
4. Present the run ID, exact target SHA, baseline, title, complete generated
   notes, plan SHA-256, and artifact names and hashes. Confirm expected merged
   pull requests are present, open pull requests are absent, and the notes
   contain nothing private or unsuitable.
5. Ask the user to explicitly approve the exact SHA, notes, and plan SHA-256.
   This is a mandatory blocking gate. Any changed approved value requires a new
   review and approval.

### 4. Publish the Approved Candidate

1. After approval, dispatch publication using only the approved preparation run
   and plan hash:

   ```sh
   gh workflow run release.yml \
     -f prepare_run_id=<run-id> \
     -f plan_sha256=<approved-plan-sha256>
   ```

2. Never create or push the stable tag locally. The workflow revalidates current
   `main`, the baseline, and tag and release absence; verifies the approved
   plan, checksums, sidecar files, and attestations; then creates the annotated
   tag and publishes only the approved candidate.

### 5. Monitor and Verify

1. Monitor the `Publish Release` workflow until completion.
2. On failure, report the failing step and relevant logs. Do not move the tag or
   replace the release automatically.
3. On success, verify that the release is stable, points to the approved SHA,
   and has the approved title and notes.
4. Verify that all three platform archives, their three `.sha256` files, and
   `checksums.txt` are attached and validate the downloaded checksums.
5. Report the release URL, tag, SHA, notes baseline, assets, verification
   results, and omitted open pull requests.

## Retry Rules

- Before publication creates the stable tag, prepare a new candidate whenever
  any approved field or current repository state changes and approve its new
  plan hash.
- Reuse the first-release baseline only when it points to the root commit.
- If the stable tag exists at the approved SHA but no release exists, stop and
  ask whether to rerun publication; do not recreate, move, or push the tag
  locally.
- If a release already exists, verify and report it instead of duplicating it.

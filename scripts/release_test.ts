import { assertEquals, assertRejects } from "@std/assert";
import {
  checkRelease,
  type CommandRunner,
  generateNotes,
  validateVersion,
} from "./release.ts";

type Reply = [string, string[], string, number?];

const mockRunner = (replies: Reply[]): CommandRunner => {
  let index = 0;
  return (command, args) => {
    const reply = replies[index++];
    if (!reply) {
      throw new Error(`Unexpected command: ${command} ${args.join(" ")}`);
    }
    assertEquals([command, args], [reply[0], reply[1]]);
    return Promise.resolve({
      code: reply[3] ?? 0,
      stdout: reply[2],
      stderr: reply[3] ? reply[2] : "",
    });
  };
};

const sha = "0123456789012345678901234567890123456789";
const baseReplies = (releases = "[]", workflow = "success"): Reply[] => [
  ["git", ["branch", "--show-current"], "main\n"],
  ["git", ["status", "--porcelain"], ""],
  ["git", ["rev-parse", "HEAD"], `${sha}\n`],
  ["git", ["rev-parse", "origin/main"], `${sha}\n`],
  [
    "git",
    ["rev-list", "--left-right", "--count", "HEAD...origin/main"],
    "0\t0\n",
  ],
  ["git", ["tag", "--list", "v1.0.0"], ""],
  ["git", [
    "ls-remote",
    "--tags",
    "origin",
    "refs/tags/v1.0.0",
    "refs/tags/v1.0.0^{}",
  ], ""],
  ["gh", [
    "release",
    "list",
    "--limit",
    "100",
    "--json",
    "tagName,isDraft,isPrerelease",
  ], releases],
  [
    "gh",
    [
      "run",
      "list",
      "--workflow",
      ".github/workflows/test.yml",
      "--commit",
      sha,
      "--limit",
      "100",
      "--json",
      "headSha,status,conclusion",
    ],
    JSON.stringify([{
      headSha: sha,
      status: "completed",
      conclusion: workflow,
    }]),
  ],
  [
    "gh",
    ["repo", "view", "--json", "nameWithOwner"],
    '{"nameWithOwner":"owner/repo"}',
  ],
];

Deno.test("validates strict semantic versions", () => {
  for (const version of ["0.0.0", "1.2.3", "1.2.3-rc.1", "1.2.3+build.4"]) {
    validateVersion(version);
  }
  for (const version of ["v1.2.3", "1.2", "01.2.3", "1.2.3-rc.01", "1.2.3-"]) {
    assertRejects(
      async () => validateVersion(version),
      Error,
      "Invalid semantic version",
    );
  }
});

Deno.test("first stable release requires root baseline and generates notes", async () => {
  const replies = baseReplies();
  replies.push(
    ["git", ["rev-list", "-n", "1", "release-notes-baseline"], `${sha}\n`],
    ["git", ["rev-list", "--max-parents=0", "HEAD"], `${sha}\n`],
    ["gh", [
      "api",
      "--method",
      "POST",
      "repos/owner/repo/releases/generate-notes",
      "-f",
      "tag_name=v1.0.0",
      "-f",
      `target_commitish=${sha}`,
      "-f",
      "previous_tag_name=release-notes-baseline",
    ], '{"name":"v1.0.0","body":"Changes"}'],
  );
  assertEquals(await generateNotes("1.0.0", mockRunner(replies)), {
    ok: true,
    version: "1.0.0",
    tag: "v1.0.0",
    title: "v1.0.0",
    body: "Changes",
    previousTag: "release-notes-baseline",
    targetSha: sha,
  });
});

Deno.test("uses latest stable release as notes baseline", async () => {
  const releases = JSON.stringify([{
    tagName: "v0.9.0",
    isDraft: false,
    isPrerelease: false,
  }]);
  assertEquals(
    (await checkRelease("1.0.0", mockRunner(baseReplies(releases))))
      .previousTag,
    "v0.9.0",
  );
});

Deno.test("rejects an existing tag", async () => {
  const replies = baseReplies().slice(0, 6);
  replies[5] = ["git", ["tag", "--list", "v1.0.0"], "v1.0.0\n"];
  await assertRejects(
    () => checkRelease("1.0.0", mockRunner(replies)),
    Error,
    "already exists",
  );
});

Deno.test("rejects a dirty worktree", async () => {
  const replies = baseReplies().slice(0, 2);
  replies[1] = ["git", ["status", "--porcelain"], " M main.ts\n"];
  await assertRejects(
    () => checkRelease("1.0.0", mockRunner(replies)),
    Error,
    "not clean",
  );
});

Deno.test("rejects a HEAD and origin/main SHA mismatch", async () => {
  const replies = baseReplies().slice(0, 4);
  replies[3] = ["git", ["rev-parse", "origin/main"], "different\n"];
  await assertRejects(
    () => checkRelease("1.0.0", mockRunner(replies)),
    Error,
    "does not match",
  );
});

Deno.test("rejects a failed Test workflow", async () => {
  const replies = baseReplies("[]", "failure").slice(0, 9);
  await assertRejects(
    () => checkRelease("1.0.0", mockRunner(replies)),
    Error,
    "has not succeeded",
  );
});

Deno.test("excludes Nightly releases from the stable baseline", async () => {
  const nightly = JSON.stringify([{
    tagName: "v0.9.0-nightly",
    isDraft: false,
    isPrerelease: false,
  }]);
  const replies = baseReplies(nightly);
  replies.push(
    ["git", ["rev-list", "-n", "1", "release-notes-baseline"], `${sha}\n`],
    ["git", ["rev-list", "--max-parents=0", "HEAD"], `${sha}\n`],
  );
  assertEquals(
    (await checkRelease("1.0.0", mockRunner(replies))).previousTag,
    "release-notes-baseline",
  );
});

Deno.test("propagates git command failures instead of returning ok", async () => {
  await assertRejects(
    () =>
      checkRelease(
        "1.0.0",
        mockRunner([[
          "git",
          ["branch", "--show-current"],
          "fatal: broken",
          128,
        ]]),
      ),
    Error,
    "git branch --show-current failed",
  );
});

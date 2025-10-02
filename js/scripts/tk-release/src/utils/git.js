import { $ } from "execa";
import semver from "semver";

export async function createCommit(appVersion, dryRun) {
  await $`git add .`;
  await $("git", ["commit", "-m", `chore: release ${appVersion}`]);
  if (!dryRun) {
    await $`git push`;
  }
}

export async function createTag(appVersion, dryRun) {
  await $`git tag ${appVersion}`;
  if (!dryRun) {
    await $`git push origin ${appVersion}`;
  }
}

export async function createBranch(appVersion, dryRun) {
  const branchRef = getBranchName(appVersion);
  if (!branchRef) {
    throw new Error(`invalid branch ref from ${appVersion}`);
  }

  await $`git branch ${branchRef}`;
  if (!dryRun) {
    await $`git push origin ${branchRef}`;
  }
}

function getBranchName(tag) {
  const sv = semver.parse(tag);
  if (!sv) return undefined;
  return `release/${sv.major}-${sv.minor}`;
}

export async function getCurrentBranch() {
  // const gitCurrentBranchProcess = await $`git branch --show-current`;
  // return gitCurrentBranchProcess.stdout;
    return "main"
}

export async function assertGit() {
  await assertGitRepo();
  // await assertGitClean();
  await assertGitBranch();
}

export async function assertGitClean() {
  const gitStatus = await $`git status --porcelain --untracked-files=all`;
  const clean = gitStatus.stdout === "";
  if (!clean) {
    throw new Error("Git is dirty. Please stash and try again.");
  }
}

export async function assertGitBranch() {
  const branch = await getCurrentBranch();
  const valid = branch === "main" || branch.startsWith("release/");

  if (!valid) {
    throw new Error(`Precondition failed: expected to be on main or release branch.`);
  }
}

export async function assertGitRepo() {
  const errMsg = "Expected execution within root of testkube-cloud-api monorepo.";
  try {
    const grep = await $`grep github.com/kubeshop/testkube go.mod`;
    if (grep.stdout.length === 0) {
      throw new Error(errMsg);
    }
  } catch (err) {
    throw new Error(errMsg);
  }
}

export async function hasGitBranch(branch) {
  const gitBranchExistProcess = await $`git ls-remote origin ${branch}`;
  return gitBranchExistProcess.stdout !== "";
}

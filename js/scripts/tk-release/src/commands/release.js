import { createCommit, createTag, createBranch } from "../utils/git.js";
import { editHelmChartOfTestkube ,editHelmChartOfRunner, parseCurrentVersion } from "../utils/helm.js";
import { failure, info, askConfirmation } from "../utils/io.js";
import semver from "semver";

export async function release({ branch, kind, dryRun, yes }) {
  const [appVersion] = parseCurrentVersion();
  const releaseStrategy = determineBumpStrategy(branch, kind, appVersion);
  const nextAppVersion = semver.inc(appVersion, releaseStrategy.release, releaseStrategy.prefix);

  console.log("Current app version:", appVersion);
  console.log("Next app version:", nextAppVersion);
  console.log("Release strategy:", releaseStrategy.release, releaseStrategy.prefix, releaseStrategy.branch);
  await inform(branch, kind, yes);

  await editHelmChartOfRunner(nextAppVersion, releaseStrategy);
  await editHelmChartOfTestkube(nextAppVersion, releaseStrategy);

  await createCommit(nextAppVersion, dryRun);
  await createTag(nextAppVersion, dryRun);

  if (releaseStrategy?.branch) {
    await createBranch(nextAppVersion, dryRun);
  }
}

async function inform(branch, kind, yes) {
  if (branch === "main") {
    switch (kind) {
      case "release":
        failure("Cannot release on main. Please first create a release candidate using the following flag: --kind=rc");
        throw new Error();
      case "rc":
        info("Creating INTERNAL release candidate.");
        break;
      case "preview":
        info("Creating EXTERNAL preview release.");
        break;
    }
  } else {
    switch (kind) {
      case "release":
        info("Creating EXTERNAL release.");
        break;
      case "rc":
        info("Creating INTERNAL release candidate.");
        break;
      case "preview":
        failure("Cannot create preview release on release branches.");
        throw new Error();
    }
  }

  await askConfirmation(yes);
}

/**
 * @returns {{release: any, prefix?: string, branch?: boolean}}
 */
function determineBumpStrategy(branch, kind, currentVersion) {
  if (branch === "main") {
    // preminor also increases the minor version
    // prerelease only increases prerelease number and keeps current minor.
    // when a prerelease already happened, then the minor bump already happend.
    const alreadyHasPreview = currentVersion.includes("-preview");
    const release = alreadyHasPreview ? "prerelease" : "preminor";

    switch (kind) {
      case "release":
        throw new Error("invalid option");
      case "rc":
        return { release, prefix: "rc", branch: true };
      case "preview":
        return { release, prefix: "preview" };
    }
  } else {
    switch (kind) {
      case "release":
        return { release: "patch" };
      case "rc":
        return { release: "prerelease", prefix: "rc" };
      case "preview":
        throw new Error("invalid option");
    }
  }
  throw new Error("invalid option");
}

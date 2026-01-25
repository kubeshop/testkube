#!/usr/bin/env node
import { release } from "./src/commands/release.js";
import { assertGit, getCurrentBranch } from "./src/utils/git.js";
import cac from "cac";
import {failure} from "./src/utils/io.js";

const cli = cac("tk-release-agent");

cli
  .command("", "Creates a release")
  .option("-k, --kind <kind>", "The kind of release. Either release, rc or preview.", { default: "release" })
  .option("-y, --yes", "Skip confirmation prompts.", { default: false })
  .option("--dry-run", "Do not push your changes. This will still commit, branch and tag.")
  .action(async (options) => {
    try {
      await assertGit();
      const kind = normaliseKind(options.kind);
      const yes = options.ci || options.yes;
      const branch = await getCurrentBranch();
      assertKind(kind, branch);

      await release({ branch, kind, dryRun: options.dryRun, yes });
    } catch (err) {
      handleErr(err);
      process.exit(1);
    }
  });

function normaliseKind(kind) {
  switch (kind) {
    case "Release":
      return "release";
    case "Release Candidate":
      return "rc";
    case "Preview Release":
      return "preview";
    default:
      return kind;
  }
}

function assertKind(kind, branch) {
  const valid = ["release", "rc", "preview"].includes(kind);
  if (!valid) throw new Error(`invalid kind: ${kind}`);

    if (branch === "main" && kind === "release") {
        failure("Cannot release on main. Please first create a release candidate using the following flag: --kind=rc");
        throw new Error();
    } else if (kind === "preview") {
        failure("Cannot create preview release on release branches.");
    }
}

cli.help();
cli.version("0.1.0");
cli.parse();

function handleErr(err) {
  if (err.stderr) {
    console.error(err.stderr);
  } else {
    console.error(err.message);
  }
}

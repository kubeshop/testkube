import { $ } from "execa";
import semver from "semver";

// Either prerelease, minor or patch.
export async function determineNextVersion(level, ref) {
  const latestTag = await getLatestTag(ref);
  if (!latestTag) {
    throw new Error("cannot determine next version: latest tag not found");
  }

  if (level === "prerelease") {
    level = latestTag?.includes("-dev") ? "prerelease" : "preminor";
  }

  const appVersion = semver.inc(latestTag, level, "rc");
  if (!appVersion) {
    throw new Error("cannot determine next version: tag is not semver");
  }
  return appVersion;
}

export async function getLatestTag(ref) {
  const $$ = $({ shell: true });
  const tagsStr = await $$`git tag --list --merged ${ref}`;
  const tags = tagsStr.stdout.split("\n");
  const filteredTags = tags.filter((t) => semver.valid(t, { loose: false }) && !t.startsWith("v"));
  const sortedTags = semver.rsort(filteredTags, { loose: false });
  return sortedTags[0];
}

export async function getLatestTagWithPattern(ref, pattern) {
  const $$ = $({ shell: true });
  const tagsStr = await $$`git tag --list --merged ${ref}`;
  const tags = tagsStr.stdout.split("\n");
  const filteredTags = tags.filter((t) => semver.valid(t, { loose: false }) && !t.startsWith("v"));
  const patternFilteredTags = filteredTags.filter(t => t.includes(pattern));
  console.log("TAGS", JSON.stringify(patternFilteredTags))
  const sortedTags = semver.rsort(patternFilteredTags, { loose: false });
  return sortedTags[0];
}
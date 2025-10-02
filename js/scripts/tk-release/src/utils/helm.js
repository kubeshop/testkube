import { info } from "./io.js";
import { $ } from "execa";
import semver from "semver";
import shell from "shelljs";

export async function editHelmChart(appVersion, releaseStrategy) {
  const subcharts = ["testkube-api", "testkube-operator", "testkube-runner", "testkube-logs"];

  for (const c of subcharts) {
    await bumpSubChartVersion(c, releaseStrategy);
    await bumpSubChartAppVersion(c, appVersion);
    await bumpSubChartValuesImageTag(c, appVersion);
  }

  await bumpChartVersion(releaseStrategy);
  await bumpChartAppVersion(appVersion);

  info("helm dependency updating…");
  await $({ cwd: "./k8s/helm/testkube" })`helm dependency update`;
}

async function bumpSubChartVersion(chart, releaseStrategy) {
  const file = `./k8s/helm/${chart}/Chart.yaml`;
  const grep = shell.grep(/^version: .*/, file);
  const current = grep.replace("version: ", "").replace("\n", "");

  const nextChartVersion = semver.inc(current, releaseStrategy.release, releaseStrategy.prefix);
  if (!nextChartVersion) return;

  shell.sed("-i", "^version:.*$", `version: ${nextChartVersion}`, file);
}

async function bumpSubChartAppVersion(chart, appVersion) {
  shell.sed("-i", "^appVersion:.*$", `appVersion: ${appVersion}`, `./k8s/helm/testkube/components/${chart}/Chart.yaml`);
}

async function bumpSubChartValuesImageTag(chart, appVersion) {
  shell.sed("-i", "^  tag:.*$", `  tag: ${appVersion}`, `./k8s/helm/testkube/components/${chart}/values.yaml`);
}

async function bumpChartVersion(releaseStrategy) {
  const file = `./k8s/helm/testkube/Chart.yaml`;
  const grep = shell.grep(/^version: .*/, file);
  const current = grep.replace("version: ", "").replace("\n", "");

  const nextChartVersion = semver.inc(current, releaseStrategy.release, releaseStrategy.prefix);
  if (!nextChartVersion) return;

  shell.sed("-i", "^version:.*$", `version: ${nextChartVersion}`, file);
}

async function bumpChartAppVersion(appVersion) {
  shell.sed("-i", "^appVersion:.*$", `appVersion: ${appVersion}`, `./k8s/helm/testkube/Chart.yaml`);
}

export function parseCurrentVersion() {
  const file = `./k8s/helm/testkube/Chart.yaml`;
  const grepAppVersion = shell.grep(/^appVersion: .*/, file);
  const appVersion = grepAppVersion.replace("appVersion: ", "").replace("\n", "");

  const grepChartVersion = shell.grep(/^version: .*/, file);
  const chartVersion = grepChartVersion.replace("version: ", "").replace("\n", "");

  return [appVersion, chartVersion];
}

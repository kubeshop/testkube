import { info } from "./io.js";
import { $ } from "execa";
import semver from "semver";
import shell from "shelljs";

export function parseCurrentVersion(chartPath = `./k8s/helm/testkube/Chart.yaml`) {
    const grepAppVersion = shell.grep(/^appVersion: .*/, chartPath);
    const appVersion = grepAppVersion.replace("appVersion: ", "").replace("\n", "");

    const grepChartVersion = shell.grep(/^version: .*/, chartPath);
    const chartVersion = grepChartVersion.replace("version: ", "").replace("\n", "");

    return [appVersion, chartVersion];
}

export async function editHelmChartOfTestkube(appVersion, releaseStrategy) {
    // Subcharts
    const apiChartPath =   `./k8s/helm/testkube-api/Chart.yaml`
    const apiValuesPath =   `./k8s/helm/testkube-api/values.yaml`
    await bumpChartVersion(apiChartPath, releaseStrategy);
    await bumpChartAppVersion(apiChartPath, appVersion);
    await bumpValuesImageTag(apiValuesPath, appVersion);

    // Remark: testkube-operator will never have new builds so do not update app version!
    const operatorChartPath =   `./k8s/helm/testkube-api/Chart.yaml`
    await bumpChartVersion(operatorChartPath, releaseStrategy);

    const chartPath = `./k8s/helm/testkube/Chart.yaml`;
    await bumpChartVersion(chartPath, releaseStrategy);
    await bumpChartAppVersion(chartPath, appVersion);

    info("helm dependency updating for testkube chart…");
    await $({ cwd: "./k8s/helm/testkube" })`helm dependency update`;
}

export async function editHelmChartOfRunner(appVersion, releaseStrategy) {
    const chartPath = `./k8s/helm/testkube-runner/Chart.yaml`;
    const valuesPath = `./k8s/helm/testkube-runner/values.yaml`;
    await bumpChartVersion(chartPath, releaseStrategy);
    await bumpChartAppVersion(valuesPath, appVersion);

    info("helm dependency updating for testkube-runner…");
    await $({ cwd: "./k8s/helm/testkube-runner" })`helm dependency update`;
}

async function bumpChartAppVersion(chartPath, appVersion) {
  shell.sed("-i", "^appVersion:.*$", `appVersion: ${appVersion}`, chartPath);
}

async function bumpValuesImageTag(valuesPath, appVersion) {
  shell.sed("-i", "^  tag:.*$", `  tag: ${appVersion}`, valuesPath);
}

async function bumpChartVersion(chartPath, releaseStrategy) {
    const grep = shell.grep(/^version: .*/, chartPath);
    const current = grep.replace("version: ", "").replace("\n", "");

    const nextChartVersion = semver.inc(current, releaseStrategy.release, releaseStrategy.prefix);
    if (!nextChartVersion) return;

    shell.sed("-i", "^version:.*$", `version: ${nextChartVersion}`, chartPath);
}


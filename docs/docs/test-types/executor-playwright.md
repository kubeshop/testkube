import Admonition from "@theme/Admonition";


# Playwright

Starting from the Testkube Helm chart version 1.9.5, it is possible to use Testkube to manage your Playwright tests inside your Kubernetes cluster.

* Default command for this executor: `<depManager>`
* Default arguments for this executor command: `<depCommand>` `playwright` `test`

Parameters in `<>` are calculated at test execution:

* `<depManager>` - `pnpm` or `npx` depending on the `DEPENDENCY_MANAGER` environment variable
* `<depCommand>` - `dlx` for `pnpm`, omitted otherwise

[See more at "Redefining the Prebuilt Executor Command and Arguments" on the Creating Test page.](../articles/creating-tests.md#redefining-the-prebuilt-executor-command-and-arguments)

export const ExecutorInfo = () => {
   return (
    <div>
      <Admonition type="info" icon="🎓" title="What is Playwright Testing?">
        <ul>
          <li><a href="https://playwright.dev/">Playwright</a> is an end-to-end testing and automation framework developed by Microsoft.</li>
          <li>Playwright supports end-to-end testing, multiple browsers, operating systems, and languages - making the work of modern developers and testers more efficient.</li>
        </ul>
      </Admonition>
    </div>
  );
}

<ExecutorInfo />


**Check out our [blog post](https://testkube.io/blog/bring-playwright-tests-into-the-cloud-with-testkube) to learn how to harness the power of Playwright Testing in your cloud-native apps.**

## Running Playwright Tests

The Playwright Testkube runner pulls the test code from Git directories. When creating a new test, this needs to be configured via the `--git-*` flags.

### Create Test

```bash
$ testkube create test --git-branch lilla/feat/playwright-executor --git-uri https://github.com/vLia/testkube-tests.git --git-path "playwright" --name playwright-test-demo --type playwright/test

Test created testkube / playwright-test-demo 🥇
```

### Run Test

```bash
$ testkube run test playwright-test-demo
Type:              playwright/test
Name:              playwright-test-demo
Execution ID:      63eb5948d2588841ffa577a0
Execution name:    playwright-test-demo-1
Execution number:  1
Status:            running
Start time:        2023-02-14 09:50:00.924165379 +0000 UTC
End time:          0001-01-01 00:00:00 +0000 UTC
Duration:          



Test execution started
Watch test execution until complete:
$ kubectl testkube watch execution playwright-test-demo-1


Use following command to get test execution details:
$ kubectl testkube get execution playwright-test-demo-1

```

To follow up with the results of the execution, you can either `watch` the execution while it is running or `get` the results of it after it is done, as seen in the commands printed out by the cli.

### Check Artifacts

To get a list of the created artifacts, use the following command:

```bash
$ testkube get artifact playwright-test-demo-1
  EXECUTION | NAME                  | SIZE (KB)  
------------+-----------------------+------------
            | playwright-report.zip |    180527  
```

These files were created and uploaded to the previously configured object storage. To download them, use the `testkube download artifact` command.

```bash
$ testkube download artifact playwright-test-demo-1 playwright-report.zip data
File data/playwright-report.zip downloaded.
```

## Special Requirements

Running tests in a containerized environment is convenient: it's simple, portable and increases the speed of development. There is a need to be aware of the limitations of this environment.

### Reports

Similarly to many other testing tools, Playwright provides the option to open a browser window for reports. It is important to make sure reporters are not opening additional windows. 

The following environment variables are set on a Dockerfile-level, but it is still important to be mindful of these differences.

```bash
ENV CI=1
ENV PWTEST_SKIP_TEST_OUTPUT=1
```

### Using Different Playwright Versions

The Testkube Playwright executor supports only one version for now: 1.30.0. In case this does not suffice, the [container executor docs](http://docs.testkube.io/test-types/container-executor/#creating-and-configuring-a-container-executor-playwright) contains instructions on how to set up your own executor with a different version of Playwright.

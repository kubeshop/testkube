# Run Tests with GitHub Actions
**The `kubeshop/testkube-run-action` has been deprecated and won't receive further updates. Use the [Testkube Action](https://github.com/marketplace/actions/testkube-action) instead.**

# Migrate from testkube-run-action to setup-testkube

1. Change the `uses` property from `kubeshop/testkube-run-action@v1` to `kubeshop/setuo-testkube@v1`.

```yaml
uses: kubeshop/setup-testkube@v1
```
2. Remove any usage of Test or Test Suite args from the `with` block.
3. Use shell scripts to run testkube CLI commands directly:
```yaml
steps:
  # Setup Testkube
  - uses: kubeshop/setup-testkube@v1
  # Pro args are still available
    with:
      organization: ${{ secrets.TkOrganization }}
      environment: ${{ secrets.TkEnvironment }}
      token: ${{ secrets.TkToken }}
  # Use CLI with a shell script
  - run: |
      # Run one or multiple testkube CLI commands, passing any arguments you need
      testkube run test some-test-name -f
```

# Deprecated usage information:

**Run on Testkube** is a GitHub Action for running tests on the Testkube platform.

Use it to run tests and test suites and obtain results directly in the GitHub's workflow.

## Usage
To use the action in your GitHub workflow, use the ``kubeshop/testkube-run-action@v1`` action. The configuration options are described in the Inputs section and may vary depending on the Testkube solution you are using (cloud or self-hosted) and your needs.

The most important options you will need are **test** and **testSuite** - you should pass a test or test suite name there.

### Testkube Pro
To use this GitHub Action for Testkube Pro, you need to create an API token.

Then, pass the **organization** and **environment** IDs for the test, along with the **token** and other parameters specific for your use case:

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  # Instance
  organization: tkcorg_0123456789abcdef
  environment: tkcenv_fedcba9876543210
  token: tkcapi_0123456789abcdef0123456789abcd
  # Options
  test: some-test-id-to-run
  ```

It will probably be unsafe to keep this directly in the workflow's YAML configuration, so you may want to use [GitHub's secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets) instead.

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  # Instance
  organization: ${{ secrets.TkOrganization }}
  environment: ${{ secrets.TkEnvironment }}
  token: ${{ secrets.TkToken }}
  # Options
  test: some-test-id-to-run
  ```

### Self-hosted Instance


To run the test on self-hosted instance, simply pass the URL that points to the API, i.e.:

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  # Instance
  url: https://demo.testkube.io/results/v1
  # Options
  test: some-test-id-to-run
  ```

### Examples

Run a test on a self-hosted instance:

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  url: https://demo.testkube.io/results/v1
  test: container-executor-curl-smoke
  ```

Run a test suite on a self-hosted instance:

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  url: https://demo.testkube.io/results/v1
  testSuite: executor-soapui-smoke-tests
  ```

Run tests from a local repository for the PR:

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  organization: ${{ secrets.TkOrganization }}
  environment: ${{ secrets.TkEnvironment }}
  token: ${{ secrets.TkToken }}
  test: e2e-dashboard-tests
  ref: ${{ github.head_ref }}
  ```

Run tests with custom environment variables:

```yaml
uses: kubeshop/testkube-run-action@v1
with:
  organization: ${{ secrets.TkOrganization }}
  environment: ${{ secrets.TkEnvironment }}
  token: ${{ secrets.TkToken }}
  test: e2e-dashboard-tests
  variables: |
    URL="https://some.domain.com"
    EXECUTED_FROM="${{ github.head_ref }}"
  secretVariables: |
    SOME_SECRET_ENV="abc"
    OTHER_ENV="${{ secrets.ExternalToken }}"
```

#### Real-life Examples
`testkube-run-action` is also used for running Testkube internal tests with Testkube. Workflow for Testkube Dashboard E2E tests can be found [here](https://github.com/kubeshop/testkube-dashboard/blob/develop/.github/workflows/pr_checks.yml#L28)

## Inputs
There are different inputs available for tests and test suites, as well as for Pro and your own instance.

### Test

```sh
| Required | Name            | Description
+----------+-----------------+-----------------------------------------------------------
|    ✓     | test            | Test name in the Testkube environment.
|    ✗     | ref             | Override Git reference (branch, commit, tag) for the test.
|    ✗     | preRunScript    | Override pre-run script for the test.
|    ✗     | variables       | Basic variables in the dotenv format.
|    ✗     | secretVariables | Secret variables in the dotenv format.
|    ✗     | executionName   | Override execution name, so you may i.e. mention the PR.
|    ✗     | namespace       | Set namespace to run test in.
```

### Test Suite

```sh
| Required | Name            | Description
+----------+-----------------+---------------------------------------------------------
|     ✓	   | testSuite	     | Test suite name in the Testkube environment.
|     ✗	   | variables	     | Basic variables in the dotenv format.
|     ✗	   | secretVariables | Variables	Secret variables in the dotenv format.
|     ✗	   | executionName   | Override execution name, so you may i.e. mention the PR.
|     ✗	   | namespace       | Set namespace to run test suite in.
```

### Pro and Pro On-Prem

```sh
| Required | Name	      | Description
+----------+--------------+------------------------------------------------------------------------------------------------------------------------------
|     ✓    | organization |	The organization ID from Testkube Pro - it starts with tkc_org, you may find it i.e. in the dashboard's URL.
|     ✓	   | environment  | The environment ID from Testkube Pro - it starts with tkc_env, you may find it i.e. in the dashboard's URL.
|     ✓	   | token        |	API token that has at least a permission to run specific test or test suite. Read more about creating API token in Testkube Pro.
|     ✗    | url          | URL of the Testkube Pro instance, if applicable.
|     ✗    | dashboardUrl | URL of the Testkube Pro dashboard, if applicable, to display links for the execution.
```

### OSS

```sh
| Required | Name         |	Description
+----------+--------------+----------------------------------------------------------------------------------------
|     ✓    | url          | URL for the API of the own Testkube instance.
|     ✗    | ws           | Override WebSocket API URL of the own Testkube instance (use it only if auto-detection doesn't work).
```

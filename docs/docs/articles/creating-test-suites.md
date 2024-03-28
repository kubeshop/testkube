# Creating Test Suites

A large IT department has a frontend team and a backend team, everything is
deployed on Kubernetes clusters, and each team is responsible for its part of the work. The frontend engineers test their code using the Cypress testing framework, but the backend engineers prefer simpler tools like Postman. They have many Postman collections defined and want to run them against a Kubernetes cluster but some of their services are not exposed externally.

A QA leader is responsible for release trains and wants to be sure that before the release all tests are completed successfully. The QA leader will need to create pipelines that orchestrate each teams' tests into a common platform.

This is easily done with Testkube. Each team can run their tests against clusters on their own, and the QA manager can create test resources and add tests written by all teams.

`Test Suites` stands for the orchestration of different test steps, which can run sequentially and/or in parallel.
On each batch step you can define either one or multiple steps such as test execution, delay, or other (future) steps.
By default the concurrency level for parallel tests is set to 10, you can redefine it using `--concurrency` option for CLI command.

## Passing Test Suite Artifacts between Steps

In some scenarios you need to access artifacts generated on previous steps of the test suite. Testkube provides two options to define which artifacts to download in the init container: all previous step artifacts or artifacts for selected steps (step number is started from 1) or artifacts for latest executions of previously executed tests (identified by names). All downloaded artifacts are stored in /data/downloaded-artifacts/{execution id} folder. See a few examples below.

## Test Suite Creation

Creating tests is really simple - create the test definition in a JSON file and pass it to the `testkube` `kubectl` plugin.

An example test file could look like this:

```sh
echo '
{
	"name": "testkube-suite",
	"description": "Testkube test suite, api, dashboard and performance",
	"steps": [
		{"execute": [{"test": "testkube-api"}, {""test": "testkube-dashboard"}]},
		{"execute": [{"delay": "1s"}]},
		{"downloadArtifacts": {"previousTestNames": ["testkube-api"]}, "execute": [{"test": "testkube-dashboard"}, {"delay": "1s"}, {""test": "testkube-homepage"}]},
		{"execute": [{"delay": "1s"}]},
		{"downloadArtifacts": {"previousStepNumbers": [1, 3]}, "execute": [{"test": "testkube-api-performance"}]},
		{"execute": [{"delay": "1s"}]},
		{"downloadArtifacts": {"allPreviousSteps": true}, "execute": [{"test": "testkube-homepage-performance"}]}
	]
}' | kubectl testkube create testsuite
```

To check if the test was created correctly, you can look at `TestSuite` Custom Resource in your Kubernetes cluster:

```sh
kubectl get testsuites -ntestkube
```

```sh title="Expected output:"
NAME                  AGE
testkube-suite           1m
testsuite-example-2   2d21h
```

To get the details of a test:

```sh
kubectl get testsuites -ntestkube testkube-suite -oyaml
```

```yaml title="Expected output:"
apiVersion: tests.testkube.io/v3
kind: TestSuite
metadata:
  creationTimestamp: "2022-01-11T07:46:12Z"
  generation: 4
  name: testkube-suite
  namespace: testkube
  resourceVersion: "57695094"
  uid: ea90a79e-bb46-49ee-a3ef-a5d99cee0a2c
spec:
  description: "Testkube test suite, api, dashboard and performance"
  steps:
  - stopOnFailure: false
    execute:
    - test: testkube-api
    - test: testkube-dashboard
  - stopOnFailure: false
    execute:
    - delay: 1s
  - stopOnFailure: false
    downloadArtifacts:
      allPreviousSteps: false
      previousTestNames:
      - testkube-api
    execute:
    - test: testkube-dashboard
    - delay: 1s
    - test: testkube-homepage
  - stopOnFailure: false
    execute:
    - delay: 1s
  - stopOnFailure: false
    downloadArtifacts:
      allPreviousSteps: false
      previousStepNumbers:
      - 1
      - 3
    execute:
    - test: testkube-api-performance
  - stopOnFailure: false
    execute:
    - delay: 1s
  - stopOnFailure: false
    downloadArtifacts:
      allPreviousSteps: true
    execute:
    - test: testkube-homepage-performance
```

Your `Test Suite` is defined and you can start running testing workflows.

## Test Suite Steps

Test Suite Steps are the individual components or actions that make up a Test Suite. They are typically a sequence of tests that are run in a specific order. There are two types of Test Suite Steps:

Tests: These are the actual tests to be run. They could be unit tests, integration tests, functional tests, etc., depending on the context.

Delays: These are time delays inserted between tests. They are used to wait for a certain period of time before proceeding to the next test. This can be useful in situations where you need to wait for some process to complete or some condition to be met before proceeding.

Similar to running a Test, running a Test Suite Step based on a test allows for specific execution request parameters to be overwritten. Step level parameters overwrite Test Suite level parameters, which in turn overwrite Test level parameters. The Step level parameters are configurable only via CRDs at the moment.

For details on which parameters are available in the CRDs, please consult the table below:

| Parameter                          | Test | Test Suite | Test Step |
| ---------------------------------- | ---- | ---------- | --------- |
| name                               | ✓    | ✓          |           |
| testSuiteName                      | ✓    |            |           |
| number                             | ✓    |            |           |
| executionLabels                    | ✓    | ✓          | ✓         |
| namespace                          | ✓    | ✓          |           |
| variablesFile                      | ✓    |            |           |
| isVariablesFileUploaded            | ✓    |            |           |
| variables                          | ✓    | ✓          |           |
| testSecretUUID                     | ✓    |            |           |
| testSuiteSecretUUID                | ✓    |            |           |
| args                               | ✓    |            | ✓         |
| argsMode                           | ✓    |            | ✓         |
| command                            | ✓    |            | ✓         |
| image                              | ✓    |            |           |
| imagePullSecrets                   | ✓    |            |           |
| sync                               | ✓    | ✓          |           |
| httpProxy                          | ✓    | ✓          | ✓         |
| httpsProxy                         | ✓    | ✓          | ✓         |
| negativeTest                       | ✓    |            |           |
| activeDeadlineSeconds              | ✓    |            |           |
| artifactRequest                    | ✓    |            |           |
| jobTemplate                        | ✓    | ✓          | ✓         |
| jobTemplateReference               | ✓    | ✓          | ✓         |
| cronJobTemplate                    | ✓    | ✓          | ✓         |
| cronJobTemplateReference           | ✓    | ✓          | ✓         |
| preRunScript                       | ✓    |            |           |
| postRunScript                      | ✓    |            |           |
| executePostRunScriptBeforeScraping | ✓    |            |           |
| sourceScripts                      | ✓    |            |           |
| scraperTemplate                    | ✓    | ✓          | ✓         |
| scraperTemplateReference           | ✓    | ✓          | ✓         |
| pvcTemplate                        | ✓    | ✓          | ✓         |
| pvcTemplateReference               | ✓    | ✓          | ✓         |
| envConfigMaps                      | ✓    |            |           |
| envSecrets                         | ✓    |            |           |
| runningContext                     | ✓    | ✓          | ✓         |
| slavePodRequest                    | ✓    |            |           |
| secretUUID                         |      | ✓          |           |
| labels                             |      | ✓          |           |
| timeout                            |      | ✓          |           |

Similar to Tests and Test Suites, Test Suite Steps can also have a field of type `executionRequest` like in the example below:

```yaml
apiVersion: tests.testkube.io/v3
kind: TestSuite
metadata:
  name: jmeter-special-cases
  namespace: testkube
  labels:
    core-tests: special-cases
spec:
  description: "jmeter and jmeterd executor - special-cases"
  steps:
  - stopOnFailure: false
    execute:
    - test: jmeterd-executor-smoke-custom-envs-replication
      executionRequest:
        args: ["-d", "-s"]
      ...
  - stopOnFailure: false
    execute:
    - test: jmeterd-executor-smoke-env-value-in-args
```

The `Definition` section of each Test Suite in the Testkube UI offers the opportunity to directly edit the Test Suite CRDs. Besides that, consider also using `kubectl edit testsuite/jmeter-special-cases -n testkube` on the command line.

### Usage Example

An example of use case for test suite step parameters would be running the same K6 load test with different arguments and memory and CPU requirements.

1. Create and Configure the Test

Let's say our test CRD stored in the file `k6-test.yaml` looks the following:

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: k6-test-parallel
  labels:
    core-tests: executors
  namespace: testkube
spec:
  type: k6/script
  content:
    type: git
    repository:
      type: git
      uri: https://github.com/kubeshop/testkube.git
      branch: main
      path: test/k6/executor-tests/
  executionRequest:
      args:
        - k6-smoke-test-without-envs.js
      jobTemplate: "apiVersion: batch/v1\nkind: Job\nspec:\n  template:\n    spec:\n      containers:\n        - name: \"{{ .Name }}\"\n          image: {{ .Image }}\n          resources:\n            requests:\n              memory: 128Mi\n              cpu: 128m\n"
      activeDeadlineSeconds: 180
```

We can apply this from the command line using:

```bash
kubectl apply -f k6-test.yaml
```

2. Run the Test

To run this test, execute:

```bash
testkube run test k6-test-parallel
```

A new Testkube execution will be created. If you investigate the new job assigned to this execution, you will see the memory and cpu limit specified in the job template was set. Checking the arguments from the `executionRequest` is also possible with:

```bash
kubectl testkube get execution k6-test-parallel-1
```

3. Create and Configure the Test Suite

We are content with the test created, but we need to make sure our application works with different kinds of loads. We could create a new Test with different parameters, but that would come with the overhead of having to manage and sync two instances of the same test. Creating a test suite makes test orchestration a more robust operation.

We have the following `k6-test-suite.yaml` file:

```yaml
apiVersion: tests.testkube.io/v3
kind: TestSuite
metadata:
  name: k6-parallel
  namespace: testkube
spec:
  description: "k6 parallel testsuite"
  steps:
  - stopOnFailure: false
    execute:
    - test: k6-test-parallel
      executionRequest:
        argsMode: override
        args:
          - -vu
          - "1"
          - k6-smoke-test-without-envs.js
        jobTemplate: "apiVersion: batch/v1\nkind: Job\nspec:\n  template:\n    spec:\n      containers:\n        - name: \"{{ .Name }}\"\n          image: {{ .Image }}\n          resources:\n            requests:\n              memory: 64Mi\n              cpu: 128m\n"
    - test: k6-test-parallel
      executionRequest:
        argsMode: override
        args:
          - -vu
          - "2"
          - k6-smoke-test-without-envs.js
```

Note that there are two steps in there running the same test. The difference is in their `executionRequest`. The first step is setting the number of virtual users to one and updating the jobTemplate to use a different memory requirement. The second test updates the VUs to 2.

Create the test suite with the command:

```bash
kubectl apply -f k6-test-suite.yaml
```

4. Run the Test Suite

Run the test suite with:

```bash
kubectl testkube run testsuite k6-parallel
```

The output of both of the test runs can be examined with:

```bash
testkube get execution k6-parallel-k6-test-parallel-2

testkube get execution k6-parallel-k6-test-parallel-3
```

The logs show the exact commands:

```bash
...
🔬 Executing in directory /data/repo:
 $ k6 run test/k6/executor-tests/k6-smoke-test-without-envs.js -vu 1
...
🔬 Executing in directory /data/repo:
 $ k6 run test/k6/executor-tests/k6-smoke-test-without-envs.js -vu 2
...
```

The job template configuration will be visible on the job level, running `kubectl get jobs -n testkube` and `kubectl get job ${job_id} -o yaml -n testkube` should be enough to check the settings.

Now we know how to increase the flexibility, reusability and scalability of your tests using test suites. By setting parameters on test suite step levels, we are making our testing automation more robust and easier to manage.

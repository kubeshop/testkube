# Creating Test Suites

A large IT department has a frontend team and a backend team, everything is
deployed on Kubernetes clusters, and each team is responsible for its part of the work. The frontend engineers test their code using the Cypress testing framework, but the backend engineers prefer simpler tools like Postman. They have many Postman collections defined and want to run them against a Kubernetes cluster but some of their services are not exposed externally.

A QA leader is responsible for release trains and wants to be sure that before the release all tests are completed successfully. The QA leader will need to create pipelines that orchestrate each teams' tests into a common platform.

This is easily done with Testkube. Each team can run their tests against clusters on their own, and the QA manager can create test resources and add tests written by all teams.

`Test Suites` stands for the orchestration of different test steps, which can run sequentially or/and in parallel.
On each batch step you can define either one or multiple steps such as test execution, delay, or other (future) steps.
By default the concurrency level for parallel tests is set to 10, you can redefine it using `--concurency` option for CLI command.

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
		{"execute": [{"test": "testkube-dashboard"}, {"delay": "1s"}, {""test": "testkube-homepage"}]},
		{"execute": [{"delay": "1s"}]},
		{"execute": [{"test": "testkube-api-performance"}]},
		{"execute": [{"delay": "1s"}]},
		{"execute": [{"test": "testkube-homepage-performance"}]}
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
    execute:
    - test: testkube-dashboard
    - delay: 1s
    - test: testkube-homepage
  - stopOnFailure: false
    execute:
    - delay: 1s
  - stopOnFailure: false
    execute:
    - test: testkube-api-performance
  - stopOnFailure: false
    execute:
    - delay: 1s
  - stopOnFailure: false
    execute:
    - test: testkube-homepage-performance
```

Your `Test Suite` is defined and you can start running testing workflows.

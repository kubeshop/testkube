# Test Suites

A large IT department has a frontend team and a backend team, everything is
deployed on Kubernetes cluster, and each team is responsible for its part of the work. The frontend engineers test their code using the  Cypress testing framework, but the backend engineers prefer simpler tools like Postman. They have a lot of Postman collections defined and want to run them against a Kubernetes cluster but some of their services are not exposed externally.

A QA leader is responsible for release trains and wants to be sure that before the release all tests are completed successfully. The QA leader will need to create pipelines that orchestrate each teams' tests into a common platform.

This is easily done with Testkube. Each team can run their tests against clusters on their own, and the QA manager can create test resources and add tests written by all teams.

`Test Suites` stands for the orchestration of different test steps such as test execution, delay, or other (future) steps.

## **Test Suite Creation**

Creating tests is really simple - create the test definition in a JSON file and pass it to the `testkube` `kubectl` plugin.

An example test file could look like this:

```sh
echo '
{
        "name": "testsuite-example-2",
        "namespace": "testkube",
        "description": "Example simple tests orchestration",
        "steps": [
                {"type": "executeTest", "namespace": "testkube", "name": "test1"},
                {"type": "delay", "duration": 5000},
                {"type": "executeTest", "namespace": "testkube", "name": "test1"}
        ]
}' | kubectl testkube create testsuite
```

To check if the test was created correctly, you can look at `TestSuite` Custom Resource in your Kubernetes cluster:

```sh
kubectl get testsuites -ntestkube

NAME                  AGE
test-example          2d21h
testsuite-example-2   2d21h
```

Get the details of a test:

```sh
kubectl get testsuites -ntestkube test-example -oyaml

apiVersion: tests.testkube.io/v1
kind: Test
metadata:
  creationTimestamp: "2022-01-11T07:46:12Z"
  generation: 4
  name: test-example
  namespace: testkube
  resourceVersion: "57695094"
  uid: ea90a79e-bb46-49ee-a3ef-a5d99cee0a2c
spec:
  description: Example simple tests orchestration
  steps:
  - execute:
      name: test1
      namespace: testkube
    type: testExecution
  - delay:
      duration: 2000
    type: delay
  - execute:
      name: test1
      namespace: testkube
    type: testExecution
```

Your `Test Suite` is defined and you can start running testing workflows.

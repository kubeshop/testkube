# Tests

Let's assume a large IT department with a frontend team and a backend team, everything is 
deployed on Kubernetes clusters, and each team is responsible for its own part of the work. The frontend engineers test their code with the use of the Cypress testing framework, but the backend engineers prefer simpler tools like Postman. They have a lot of Postman collections defined and want to run them against Kubernetes cluster, but, unfortunately, some of their services are not exposed externally.

The Quality Assurance manager is responsible for release success and needs to be sure that all tests are successful before the release. To do this with the above multiple tool testing process, pipelines must be created to orchestrate each team's tests into some common platform. 

TestKube simplifies this process. Each team can run their tests against clusters easily on their own, and the QA manager can create test resources and add test scripts written by all teams.  

`Tests` stands for orchestration, orchestration of different test steps like script execution and delay, or other (future) steps. 

# **Tests Creation**

Creating tests is really simple. Define the test in a JSON file and then pass it to the `testkube` `kubectl` plugin.

An example test file might look like this: 

```sh
echo '
{
        "name": "test-example-2",
        "namespace": "testkube",
        "description": "Example simple test orchestration",
        "steps": [
                {"type": "executeScript", "namespace": "testkube", "name": "test1"},
                {"type": "delay", "duration": 5000},
                {"type": "executeScript", "namespace": "testkube", "name": "test1"}
        ]
}' | kubectl testkube tests create
```

To check if the test was created correctly, look at the `Test Custom Resource` in your Kubernetes cluster: 
```sh
kubectl get tests -ntestkube

NAME             AGE
test-example     2d21h
test-example-2   2d21h
```

The details of the test: 
```sh 
kubectl get tests -ntestkube test-example -oyaml

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
  description: Example simple test orchestration
  steps:
  - execute:
      name: test1
      namespace: testkube
    type: scriptExecution
  - delay:
      duration: 2000
    type: delay
  - execute:
      name: test1
      namespace: testkube
    type: scriptExecutio
```

Your `Test` is now defined and you can start running testing workflows. 

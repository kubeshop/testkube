# TestSuites

Let's assume a quite big IT department where there is a frontend team and backend team, everything is 
deployed on Kubernetes cluster, and each team is responsible for their part of work. Frontend engineers test their code with the use of Cypress testing framework, but backend engineers prefer simpler tools like Postman, they have a lot of Postman collections defined and want to run them against Kubernetes cluster, unfortunately, some of their services are not exposed externally.

There is also some QA leader who is responsible for release trains and wants to be sure that before release all tests are green. The one issue is that he needs to create pipelines that orchestrate all teams tests into some common platform. 

... it would be so easy if all of them have used Testkube. Each team can run their tests against clusters easily on their own, and the QA manager can create Test resources and add test tests written by all teams.  

`TestSuites` stand for orchestration, orchestration of different test steps like e.g. test execution, delay, or other (future) steps. 
# TestSuites creation

Creating tests is really simple - you need to write down test definition in json file and then pass it to `testkube` `kubectl` plugin.

example test file could look like this: 

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
}' | kubectl testkube testsuites create
```

To check if test was created correctly you can look at `TestSuite` Custom Resource in your Kubernetes cluster: 
```sh
kubectl get testssuites -ntestkube

NAME                  AGE
test-example          2d21h
testsuite-example-2   2d21h
```

and get details of some test: 
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

Your `TestSuite` from now is defined, you can start running testing workflows from now. 

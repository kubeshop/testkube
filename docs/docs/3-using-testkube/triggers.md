---
sidebar_position: 9
sidebar_label: Test Triggers
---
# Test Triggers

Testkube allows you to automate running tests and test suites by defining triggers on certain events for various
Kubernetes resources.

## **Architecture**

Testkube uses [informers](https://pkg.go.dev/k8s.io/client-go/informers) to watch Kubernetes resources and register handlers
on certain actions on the watched Kubernetes resources.

Informers are a reliable, scalable and fault-tolerant Kubernetes concept where each informer registers handlers with the
Kubernetes API and gets notified by Kubernetes on each event on the watched resources.

## **Model**

### Schema

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: Kubernetes object name
  namespace: Kubernetes object namespace
spec:
  resource: for which Resource do we monitor Event which triggers an Action
  resourceSelector: resourceSelector identifies which Kubernetes objects should be watched
  event: on which Event for a Resource should an Action be triggered
  action: action represents what needs to be executed for selected execution
  execution: execution identifies for which test execution should an action be executed
  testSelector: testSelector identifies on which Testkube Kubernetes Objects an action should be taken
```

**resourceSelector** and **testSelector** support selecting resources by name or using
Kubernetes [Label Selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#resources-that-support-set-based-requirements)

```
selector := resourceSelector | testSelector
```

Selecting resources by name:
```yaml
selector:
  name: Kubernetes object name
  namespace: Kubernetes object namespace (default is **testkube**)
```

Selecting resources using Label Selector:
```yaml
selector:
  namespace: Kubernetes object namespace (default is **testkube**)
  labelSelector:
    matchLabels: map of key-value pairs
    matchExpressions: "array of key: string, operator: string and values: []string objects"
```

Supported values:
* **resource**  - pod, deployment, statefulset, daemonset, service, ingress, event
* **action**    - run
* **event**     - created, modified, deleted
* **execution** - test, testsuite

**NOTE**: all resources support the above-mentioned events, a list of finer-grained events is in the works, stay tuned...

### Example

Example which creates a test trigger with the name **testtrigger-example** in the **default** namespace for **pods**
which have the **testkube.io/tier: backend** label which gets triggered on **modified** event and **runs** a **testsuite**
identified by the name **sanity-test** in the **frontend** namespace

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: testtrigger-example
  namespace: default
spec:
  resource: pod
  resourceSelector:
    labelSelector:
      matchLabels:
        testkube.io/tier: backend
  event: modified
  action: run
  execution: testsuite
  testSelector:
    name: sanity-test
    namespace: frontend
```

## API

Testkube exposes CRUD operations on test triggers in the REST API. Check out the OpenAPI docs for more info.

---
sidebar_position: 9
sidebar_label: Test Triggers
---
# Test Triggers

Testkube allows you to automate running tests and test suites by defining triggers on certain events for various
Kubernetes resources.

In generic terms, a **trigger** defines an **action** which will be executed for a given **execution** when a certain **event** on a specific **resource** occurs.

For example, we could define a **trigger** which **runs** a **test** when a **configmap** gets **modified**.

## Custom Resource Definition Model

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
  conditionSpec: which resource conditions should be matched
  action: action represents what needs to be executed for selected execution
  execution: execution identifies for which test execution should an action be executed
  delay: "OPTIONAL: add a delay before scheduling a test or testsuite when a trigger is matched to an event"
  testSelector: testSelector identifies on which Testkube Kubernetes Objects an action should be taken
```

### Selectors

**resourceSelector** and **testSelector** fields support selecting resources either by name or using
Kubernetes [Label Selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#resources-that-support-set-based-requirements).

Each selector should specify the **namespace** of the object, otherwise the namespace defaults to **testkube**.

```
selector := resourceSelector | testSelector
```

#### Name Selector

Name selectors are used when we want to select a specific resource in a specific namespace.

```yaml
selector:
  name: Kubernetes object name
  namespace: Kubernetes object namespace (default is **testkube**)
```

#### Label Selector

Label selectors are used when we want to select a group of resources in a specific namespace.

```yaml
selector:
  namespace: Kubernetes object namespace (default is **testkube**)
  labelSelector:
    matchLabels: map of key-value pairs
    matchExpressions:
      - key: label name
        operator: one of In, NotIn, Exists, DoesNotExist
        values: list of values
```

### Resource Conditions

Resource Conditions allows triggers to be defined based on the status conditions for a specific resource.

```yaml
conditionSpec:
    timeout: duration in seconds the test trigger waits for conditions, until its stopped
    conditions:
    - type: test trigger condition type
      status: test trigger condition status, supported values - True, False, Unknown
      reason: test trigger condition reason
```

### Supported values
* **resource**  - pod, deployment, statefulset, daemonset, service, ingress, event, configmap
* **action**    - run
* **event**     - created, modified, deleted
* **execution** - test, testsuite

**NOTE**: All resources support the above-mentioned events, a list of finer-grained events is in the works, stay tuned...

## Example

Here is an example for a **Test Trigger** *default/testtrigger-example* which runs the **TestSuite** *frontend/sanity-test*
when a **pod** containing the label **testkube.io/tier: backend** gets **modified** and also has the conditions **Progressing: True: NewReplicaSetAvailable** and **Available: True**.

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
  conditionSpec:
    timeout: 100
    conditions:
    - type: Progressing
      status: "True"
      reason: "NewReplicaSetAvailable"
    - type: Available
      status: "True"
  action: run
  execution: testsuite
  testSelector:
    name: sanity-test
    namespace: frontend
```

## Architecture

Testkube uses [Informers](https://pkg.go.dev/k8s.io/client-go/informers) to watch Kubernetes resources and register handlers
on certain actions on the watched Kubernetes resources.

Informers are a reliable, scalable and fault-tolerant Kubernetes concept where each informer registers handlers with the
Kubernetes API and gets notified by Kubernetes on each event on the watched resources.

## API

Testkube exposes CRUD operations on test triggers in the REST API. Check out the [Open API](6-openapi.md) docs for more info.

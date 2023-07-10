# Triggers

Testkube allows you to automate running tests and test suites by defining triggers on certain events for various Kubernetes resources.

## What is a Testkube Test Trigger?

In generic terms, a _Trigger_ defines an _action_ which will be executed for a given _execution_ when a certain _event_ on a specific _resource_ occurs. For example, we could define a _TestTrigger_ which _runs_ a _Test_ when a _ConfigMap_ gets _modified_.

Watch our [video guide](#video-tutorial) on using Testkube Test Triggers to perform **Asynchronous Testing in Kubernetes**:


## Custom Resource Definition Model
### Selectors

The `resourceSelector` and `testSelector` fields support selecting resources either by name or using
the Kubernetes [Label Selector](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#resources-that-support-set-based-requirements).

Each selector should specify the `namespace` of the object, otherwise the namespace defaults to `testkube`.

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
        operator: [In | NotIn | Exists | DoesNotExist
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
      ttl: test trigger condition ttl
```

### Supported Values
* **Resource**  - pod, deployment, statefulset, daemonset, service, ingress, event, configmap
* **Action**    - run
* **Event**     - created, modified, deleted
* **Execution** - test, testsuite

**NOTE**: All resources support the above-mentioned events, a list of finer-grained events is in the works, stay tuned...

## Example

Here is an example for a **Test Trigger** *default/testtrigger-example* which runs the **TestSuite** *frontend/sanity-test*
when a **deployment** containing the label **testkube.io/tier: backend** gets **modified** and also has the conditions **Progressing: True: NewReplicaSetAvailable** and **Available: True**.

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: testtrigger-example
  namespace: default
spec:
  resource: deployment
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
      ttl: 60
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

Testkube exposes CRUD operations on test triggers in the REST API. Check out the [Open API](../openapi.md) docs for more info.

## Video Tutorial 

<iframe width="100%" height="350px" src="https://www.youtube.com/embed/t4V6E9rQ5W4" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>


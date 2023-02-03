---
sidebar_position: 2
sidebar_label: Custom Resources
---
# Testkube Custom Resources

In Testkube, Tests, Test Suites, Executors and Webhooks are defined using [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). The current definitions can be found in the [kubeshop/testkube-operator](https://github.com/kubeshop/testkube-operator/tree/main/config/crd) repository.

You can always check the list of all CRDs using `kubectl` configured to point to your Kubernetes cluster with Testkube installed:

```bash
$ kubectl get crds
NAME                                  CREATED AT
certificaterequests.cert-manager.io   2022-04-01T10:53:54Z
certificates.cert-manager.io          2022-04-01T10:53:54Z
challenges.acme.cert-manager.io       2022-04-01T10:53:54Z
clusterissuers.cert-manager.io        2022-04-01T10:53:54Z
executors.executor.testkube.io        2022-04-13T11:44:22Z
issuers.cert-manager.io               2022-04-01T10:53:54Z
orders.acme.cert-manager.io           2022-04-01T10:53:54Z
scripts.tests.testkube.io             2022-04-13T11:44:22Z
tests.tests.testkube.io               2022-04-13T11:44:22Z
testsuites.tests.testkube.io          2022-04-13T11:44:22Z
webhooks.executor.testkube.io         2022-04-13T11:44:22Z
```

To check details on one of the CRDs, use `describe`:

```bash
$ kubectl describe crd tests.tests.testkube.io
Name:         tests.tests.testkube.io
Namespace:    
Labels:       app.kubernetes.io/managed-by=Helm
Annotations:  controller-gen.kubebuilder.io/version: v0.4.1
              meta.helm.sh/release-name: testkube
              meta.helm.sh/release-namespace: testkube
API Version:  apiextensions.k8s.io/v1
Kind:         CustomResourceDefinition
...
```

Below, you will find short descriptions and example declarations of the custom resources defined by Testkube.

## **Tests**

Testkube Tests can be defined as a single executable unit of tests. Depending on the test type, this can mean one or multiple test files.

To get all the test types available in your cluster, check the executors:

```bash
$ kubectl testkube get executors -o yaml | grep -A1 types
    types:
    - postman/collection
--
    types:
    - curl/test
--
    types:
    - cypress/project
--
    types:
    - k6/script
--
    types:
    - postman/collection
--
    types:
    - soapui/xml
```

When creating a Testkube Test, there are multiple supported input types:

* String
* Git directory
* Git file
* File URI

Variables can be configured using the `variables` field as shown below.

```yml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: example-test
  namespace: testkube
spec:
  content:
    data: "{...}"
    type: string
  type: postman/collection
  executionRequest:
    variables:
      var1:
        name: var1
        type: basic
        value: val1
      sec1:
        name: sec1
        type: secret
        valueFrom:
          secretKeyRef:
            key: sec1
            name: vartest4-testvars
```

## **Test Suites**

Testkube Test Suites are collections of Testkube Tests of the same or different types.

```yml
apiVersion: tests.testkube.io/v2
kind: TestSuite
metadata:
  name: example-testsuite
  namespace: testkube
spec:
  description: Example Test Suite
  steps:
    - execute:
        name: example-test1
        namespace: testkube
    - delay:
        duration: 1000
    - execute:
        name: example-test2
        namespace: testkube
```

## **Executors**

Executors are Testkube-specific test runners. There are a list of predefined Executors coming with Testkube. You can also write your own custom Testkube Executor using [this guide](https://kubeshop.github.io/testkube/executor-custom/).

Example:

```yml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: example-executor
  namespace: testkube
spec:
  executor_type: job  
  image: YOUR_USER/testkube-executor-example:1.0.0 
  types:
  - example/test      
  content_types:
  - string
  - file-uri
  - git-file
  - git-dir
  features: 
  - artifacts
  - junit-report
  meta:
   iconURI: http://mydomain.com/icon.jpg
   docsURI: http://mydomain.com/docs
   tooltips:
    name: please enter executor name
```

## **Webhooks**

Testkube Webhooks are HTTP POST calls having the Testkube Execution object and its current state as payload. They are sent when a test is either started or finished. This can be defined under `events`.

```yml
apiVersion: executor.testkube.io/v1
kind: Webhook
metadata:
  name: example-webhook
  namespace: testkube
spec:
  uri: http://localhost:8080/events
  events:
  - start-test
  - end-test
```

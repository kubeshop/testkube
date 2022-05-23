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

Below, you will find short descriptions and minimal example definitions of the custom resources defined by Testkube.

## Tests

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

* string
* file path
* Git directory
* Git file
* File URI

Parameters and secrets can also be configured.

```yml
apiVersion: tests.testkube.io/v2
kind: Test
metadata:
    type: "soapui/xml"
spec:
    content:
        data: <xml>...</xml>
        type: string
    type: "soapui/xml"
```

## Test Suites

Testkube Test Suites are collections of Testkube Tests of the same or different types.

```yml

```

## Executors

Executors are Testkube-specific test runners.

Example:

```yml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: example-executor
  namespace: testkube
spec:
  executor_type: job  
  # 'job' is currently the only type for custom executors
  image: YOUR_USER/testkube-executor-example:1.0.0 
  # pass your repository and tag
  types:
  - example/test      
  # your custom type registered (used when creating and running your testkube tests)
  content_types:
  - string             # test content as string
  - file-uri           # http based file content
  - git-file           # file stored in Git
  - git-dir            # entire dir/project stored in Git
  features: 
  - artifacts          # executor can have artifacts after test run (e.g. videos, screenshots)
  - junit-report       # executor can have junit xml based results
```

## Webhooks

Webhooks are ???

```yml

```

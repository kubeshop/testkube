# Testkube Custom Resources

In Testkube, Tests, Test Suites, Executors and Webhooks, Test Sources and Test Triggers are defined using [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). The current definitions can be found in the [kubeshop/testkube-operator](https://github.com/kubeshop/testkube-operator/tree/main/config/crd) repository.

You can always check the list of all CRDs using `kubectl` configured to point to your Kubernetes cluster with Testkube installed:

```sh
kubectl get crds -n testkube
```

```sh title="Expected output:"
NAME                                  CREATED AT
executors.executor.testkube.io        2023-06-15T14:49:11Z
scripts.tests.testkube.io             2023-06-15T14:49:11Z
templates.tests.testkube.io           2023-06-15T14:49:11Z
testexecutions.tests.testkube.io      2023-06-15T14:49:11Z
tests.tests.testkube.io               2023-06-15T14:49:11Z
testsources.tests.testkube.io         2023-06-15T14:49:11Z
testsuiteexecutions.tests.testkube.io 2023-06-15T14:49:11Z
testsuites.tests.testkube.io          2023-06-15T14:49:11Z
testtriggers.tests.testkube.io        2023-06-15T14:49:11Z
webhooks.executor.testkube.io         2023-06-15T14:49:11Z
```

To check details on one of the CRDs, use `describe`:

```sh
kubectl describe crd tests.tests.testkube.io
```

```sh title="Expected output:"
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

You can find the description of each CRD in the [CRDs Reference](./crds-reference.md) section of the documentation.

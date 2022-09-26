---
sidebar_position: 11
sidebar_label: Container Executor
---
# What is a Container Executor?

The TestKube Container Executor allows you to run your own container images for executing tests. TestKube orchestrates the Tests using the container image as Kubernetes Jobs.

The Test execution fails if the container exits with an error and succeeds when the container command successfully executes.

In order to use the Container Executor, create a new executor with `executor_type: container` and your custom type. For example:

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: curl-container-executor
  namespace: testkube
spec:
  image: curlimages/curl:7.85.0
  command: ["curl"]
  executor_type: container
  imagePullSecrets:
    - name: secret-name
  types:
  - curl-container/test
```

In the above example, all Tests of `curl-container/test` will be executed by this Executor. Then you can create a new test that uses this Executor:

```
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: test-website
  namespace: testkube
spec:
  type: curl-container/test
  executionRequest:
    args:
    - https://testkube.kubeshop.io/
    envs:
      TESTKUBE_ENV: example
```

Executing this test will run a Kubernetes Job with a `curlimages/curl:7.85.0` image, `secret-name` image pull secret, `curl` command, and `https://testkube.kubeshop.io/` argument.

You can provide image, args, command, and image pull secrets in the HTTP Request, Test Spec, and Executor spec. The container executor merges all the data using the following order:

1. HTTP Request.
2. Test.Spec.ExecutionRequest fields are used if they are not filled before.
3. Executor.Spec fields are used if they are not filled before.

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

```yaml
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

Executing this test will run a Kubernetes Job with:
- `curlimages/curl:7.85.0` image
- `secret-name` image pull secret
- `curl` command
- `https://testkube.kubeshop.io/` argument

You can provide image, args, command, and image pull secrets in the HTTP Request, Test Spec, and Executor spec. The container executor merges all the data using the following order:

1. HTTP Request.
2. Test.Spec.ExecutionRequest fields are used if they are not filled before.
3. Executor.Spec fields are used if they are not filled before.

## Input Data

You can provide input data via string, files, and Git repositories via TestKube Dashboard. The data is downloaded into `/data` before the test is run using Kubernetes Init container. When using `string` type, the content is put into `/data/test-content` file. For example:

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: custom-container
  namespace: testkube
spec:
  content:
    data: |-
      {
        "project": "testkube",
        "is": "awesome"
      }
    type: string
  type: custom-container/test
```

Puts data into `/data/test-content` file:

```bash
$ cat /data/test-content
{
  "project": "testkube",
  "is": "awesome"
}
```

When using Git or Git directory sources, the content is placed inside `/data/repo` directory. For example:

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: custom-container
  namespace: testkube
spec:
  content:
    repository:
      branch: main
      type: git-dir
      uri: https://github.com/kubeshop/testkube-executor-init
    type: git-dir
```

Downloads into `/data/repo` directory

```bash
$ ls /data/repO
CODE_OF_CONDUCT.md  CONTRIBUTING.md  LICENSE  Makefile  README.md  build  cmd  go.mod  go.sum  pkg
```
## Collecting test artifacts
For container executors that produce files during test execution we support collecting (scraping) these artifacts and storing them in our S3 compatible file storage. You need to save test related files into specified directories on the dynamically created volume, they will be uploaded from there to Testkube file storage and available later for downloading using standard CLI or UI commands. For example:

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: cli-container
  namespace: testkube
spec:
  type: cli/container
  executionRequest:
    artifactRequest:
      storageClassName: standard
      volumeMountPath: /share
      dirs:
      - test/reports
```

You have to define the storage class name, volume mount path and directories in this volume with test artifacts.
Make sure your container executor definition has `artifacts` feature. For example:

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: cli-container-executor
  namespace: testkube
spec:
  types:
  - cli/container
  executor_type: container
  image: soleware/nx-cli:8.5.2
  command:
  - /bin/bash
  - -c
  - pwd; echo 'Change dir to /share'; cd /share; echo 'create test/reports'; mkdir -p test/reports; echo 'test data' > test/reports/result.txt
  features:
  - artifacts

```

Run your test using CLI command:

```bash
kubectl testkube run test cli-container
```

Then get available artifacts for your test execution id:

```bash
kubectl testkube get artifact 638a08b94ff1d2c694aeebf2
```

Output:

```bash
  NAME       | SIZE (KB)  
-------------+------------
  result.txt |        10  
```

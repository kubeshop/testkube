# Migrating Tests, Test Suites and Executors to Test Workflows and Test Workflow Templates

## Introduction

In order to simplify migration from Tests and Test Suites for Test Workflows you can
use `kubectl testkube migrate` Testkube CLI command. It generates CRD definitions for 
Test Workflows and Test Workflow Tempaltes using your existing Test, Test Suite and Executor
resources. You will need to check prepared CRDs and apply them to your Kubernetes cluster.
This command contains 2 subcommands: `test` and `testsuite`. It's possible to migrate all
existing resources or only particular resource.

## Test Migration

Tests are migrated to Test Workflows, by default Test Workflow Template is generated for 
the Executor connected to the Test. Testkube Test Workflows are not generated for official
Testkube Test Workflow Templates (K6, Postman, Cypress and Playwright ones)

### Example - Test Workflow for Pre-built Postman Test

Original Test CRD

```yaml
apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: postman-executor-smoke
  namespace: testkube
  labels:
    core-tests: executors
    executor: postman-executor
    test-type: postman-collection
spec:
  type: postman/collection
  content:
    type: git
    repository:
      type: git
      uri: https://github.com/kubeshop/testkube.git
      branch: main
      path: test/postman/executor-tests/postman-executor-smoke.postman_collection.json
  executionRequest:
    args:
      - "--env-var"
      - "TESTKUBE_POSTMAN_PARAM=TESTKUBE_POSTMAN_PARAM_value"
```

```sh
kubectl testkube migrate test postman-executor-smoke
```

Resulted Test Workflow CRD

```yaml
kind: TestWorkflow
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: postman-executor-smoke
  namespace: testkube
  labels:
    core-tests: executors
    executor: postman-executor
    test-type: postman-collection
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube.git
      revision: main
      paths:
      - test/postman/executor-tests/postman-executor-smoke.postman_collection.json
  job:
    labels:
      core-tests: executors
      executor: postman-executor
      test-type: postman-collection
  steps:
  - name: Run tests
    template:
      name: official--postman--beta
      config:
        run: newman run --env-var TESTKUBE_POSTMAN_PARAM=TESTKUBE_POSTMAN_PARAM_value
          /data/repo/test/postman/executor-tests/postman-executor-smoke.postman_collection.json
```

### Example - Test Workflow and Test Workflow Template for K6 Container Executor Test

Original Test and Executor CRD

```yaml
apiVersion: executor.testkube.io/v1
kind: Executor
metadata:
  name: container-executor-k6-0.43.1
  namespace: testkube
spec:
  types:
  - container-executor-k6-0.43.1/test
  executor_type: container
  image: grafana/k6:0.43.1

---

apiVersion: tests.testkube.io/v3
kind: Test
metadata:
  name: container-executor-k6-smoke
  namespace: testkube
  labels:
    core-tests: executors
    executor: container-executor-k6-0.43.1
    test-type: container-executor-k6-0-43-1-test
spec:
  type: container-executor-k6-0.43.1/test
  content:
    type: git
    repository:
      type: git
      uri: https://github.com/kubeshop/testkube
      branch: main
      path: test/k6/executor-tests/k6-smoke-test-without-envs.js
      workingDir: test/k6/executor-tests
  executionRequest:
    args:
      - "run"
      - "k6-smoke-test-without-envs.js"
    activeDeadlineSeconds: 180
```

```sh
kubectl testkube migrate test container-executor-k6-smoke
```

Resulted Test Workflow and Test Workflow Template CRD

```yaml
kind: TestWorkflowTemplate
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: container-executor-k6-0.43.1
  namespace: testkube
spec:
  container:
    image: grafana/k6:0.43.1
  pod: {}

---

kind: TestWorkflow
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: container-executor-k6-smoke
  namespace: testkube
  labels:
    core-tests: executors
    executor: container-executor-k6-0.43.1
    test-type: container-executor-k6-0-43-1-test
spec:
  use:
  - name: container-executor-k6-0.43.1
  content:
    git:
      uri: https://github.com/kubeshop/testkube
      revision: main
      paths:
      - test/k6/executor-tests/k6-smoke-test-without-envs.js
  container:
    workingDir: /data/repo/test/k6/executor-tests
  job:
    labels:
      core-tests: executors
      executor: container-executor-k6-0.43.1
      test-type: container-executor-k6-0-43-1-test
    activeDeadlineSeconds: 180
  steps:
  - name: Run tests
    run:
      args:
      - run
      - k6-smoke-test-without-envs.js
```

## Test Suite Migration

Test Suites are migrated to Test Workflows, by default Test Workflows are not generated for 
the Tests used in the Test Suites.

Original Test Suite CRD

```yaml
apiVersion: tests.testkube.io/v3
kind: TestSuite
metadata:
  name: executor-container-k6-smoke-tests
  namespace: testkube
  labels:
    core-tests: executors
spec:
  description: "container executor k6 smoke tests"
  steps:
  - stopOnFailure: false
    execute:
    - test: k6-executor-smoke

```

```sh
kubectl testkube migrate testsuite executor-container-k6-smoke-tests
```

Resulted Test Workflow CRD

```yaml
kind: TestWorkflow
apiVersion: testworkflows.testkube.io/v1
metadata:
  name: executor-container-k6-smoke-tests
  namespace: testkube
  labels:
    core-tests: executors
description: container executor k6 smoke tests
spec:
  job:
    labels:
      core-tests: executors
  steps:
  - name: Run test workflows
    optional: true
    steps:
    - name: Run tests
      execute:
        workflows:
        - name: k6-executor-smoke
```
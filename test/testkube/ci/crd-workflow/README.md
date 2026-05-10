# API server CI TestWorkflow/TestTrigger usage

This directory contains a ready-to-apply TestWorkflow + TestTrigger pair for running `lint` and `build` for `cmd/api-server` when repository content changes on `main`.

- Workflow: `api-server-build-lint.yaml`
- Trigger: `api-server-build-lint-trigger.yaml`
- Namespace: `testkube`

## Prerequisites

- Testkube installed
- `TestkubeNamespace` configured as `testkube`
- Access to apply CRDs in `testkube` namespace

## Quick start

```bash
kubectl apply -n testkube -f ./api-server-build-lint.yaml
kubectl apply -n testkube -f ./api-server-build-lint-trigger.yaml
```

## Example 1: Trigger on any change in `main` (current config)

The included trigger watches:

- `uri`: `https://github.com/kubeshop/testkube`
- `revision`: `main`
- any content change (no path filter)

This is the default behavior in `api-server-build-lint-trigger.yaml`.

## Example 2: Trigger only when `cmd/api-server/**` changes

If you want to reduce executions, add path filtering:

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: api-server-ci-main-content-trigger
  namespace: testkube
spec:
  resource: content
  event: modified
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      revision: main
      paths:
        - cmd/api-server/**
  action: run
  execution: testworkflow
  testSelector:
    name: api-server-ci-build-lint
    namespace: testkube
```

## Example 3: Run against a different branch

Use a different revision in both workflow content and trigger content selector:

```yaml
spec:
  content:
    git:
      uri: https://github.com/kubeshop/testkube
      revision: release/v2
```

```yaml
spec:
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      revision: release/v2
```

## Example 4: Manual run without trigger

You can run the workflow manually for verification:

```bash
kubectl testkube run testworkflow api-server-ci-build-lint -n testkube
```

## Verify resources

```bash
kubectl get testworkflow api-server-ci-build-lint -n testkube
kubectl get testtrigger api-server-ci-main-content-trigger -n testkube
```

## Cleanup

```bash
kubectl delete -n testkube -f ./api-server-build-lint-trigger.yaml
kubectl delete -n testkube -f ./api-server-build-lint.yaml
```

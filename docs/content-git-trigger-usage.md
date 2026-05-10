# ContentGit usage for TestTrigger and WorkflowTrigger

This guide shows how to configure git-based content triggers using the new `ContentGit` fields.

## TestTrigger: watch any change in a branch

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: trigger-on-main-changes
  namespace: testkube
spec:
  resource: content
  event: modified
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      revision: main
  action: run
  execution: testworkflow
  testSelector:
    name: my-workflow
    namespace: testkube
```

## TestTrigger: watch selected paths and use credentials from Secret

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: trigger-api-only
  namespace: testkube
spec:
  resource: content
  event: modified
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      revision: main
      authType: token
      tokenFrom:
        secretKeyRef:
          name: git-creds
          key: token
      paths:
        - cmd/api-server/**
        - pkg/triggers/**
  action: run
  execution: testworkflow
  testSelector:
    name: my-workflow
    namespace: testkube
```

## WorkflowTrigger: watch any content change

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: WorkflowTrigger
metadata:
  name: workflow-trigger-main
  namespace: testkube
spec:
  when:
    event: modified
    git:
      uri: https://github.com/kubeshop/testkube
      revision: main
  run:
    workflow:
      name: my-workflow
```

## WorkflowTrigger: watch selected paths with SSH key from Secret

```yaml
apiVersion: testworkflows.testkube.io/v1
kind: WorkflowTrigger
metadata:
  name: workflow-trigger-ssh
  namespace: testkube
spec:
  when:
    event: modified
    git:
      uri: git@github.com:kubeshop/testkube.git
      revision: main
      authType: ssh
      sshKeyFrom:
        secretKeyRef:
          name: git-creds
          key: ssh-private-key
      paths:
        - test/**
        - cmd/**
  run:
    workflow:
      name: my-workflow
```

## Notes

- `paths` is a change filter. If omitted, all repository paths are watched.
- `revision` accepts a branch, tag, or commit SHA.
- Prefer `tokenFrom` / `sshKeyFrom` (and `usernameFrom`) over inline plain-text fields.

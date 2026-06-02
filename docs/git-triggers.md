# Git Event Triggers

Git Event Triggers let you run Test Workflows when files in a Git repository change. They are useful when your test inputs live in Git and you want Testkube to react automatically without depending on an external CI system.

Typical use cases include:

- running regression or smoke workflows when application code changes in a specific branch
- validating API, contract, or configuration changes only when selected folders are updated
- re-running workflows when shared test data, manifests, or reusable test assets are updated

This can be a good alternative to GitHub Actions when you want both the trigger and the execution lifecycle to stay inside Testkube.

## How Git Event Triggers work

For each trigger, Testkube polls the configured repository and revision on a configurable interval (1 minute by default). On each poll, it fetches the current state of the target repository, applies any configured `paths` filters, and compares the current HEAD commit with the last cached commit for that trigger. When Testkube detects a new matching change, it emits a content `modified` event and runs the selected workflow.

This guide shows how to configure git-based content triggers using the `ContentGit` fields.

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
      authType: basic
      tokenFrom:
        secretKeyRef:
          name: git-creds
          key: token
      paths:
        - cmd/api-server
        - pkg/triggers
  action: run
  execution: testworkflow
  testSelector:
    name: my-workflow
    namespace: testkube
```

## Notes

- `paths` is a change filter. If omitted, all repository paths are watched.
- `paths` supports exact paths or directory/file prefixes (`path` or `path/...` semantics), not glob patterns.
- `revision` accepts a branch, tag, or commit SHA. For triggers that should observe future changes, use a moving ref such as a branch or tag. A commit SHA is a pinned, immutable revision and is valid only as a fixed baseline; it will not observe future changes.
- The polling interval is configurable; by default Testkube checks the repository every 1 minute.
- Testkube caches the last-seen HEAD commit per trigger in memory to detect new changes between polling cycles.
- After API server restart or leader failover, each trigger is re-baselined to the current HEAD and commits pushed while the informer was down are not replayed.
- Prefer `tokenFrom` / `sshKeyFrom` (and `usernameFrom`) over inline plain-text fields.
- SSH auth requires host key verification via `known_hosts` (for example by mounting a known_hosts file and setting `SSH_KNOWN_HOSTS`).

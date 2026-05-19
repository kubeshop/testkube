# ContentGit usage for TestTrigger

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
- Prefer `tokenFrom` / `sshKeyFrom` (and `usernameFrom`) over inline plain-text fields.
- SSH auth requires host key verification via `known_hosts` (for example by mounting a known_hosts file and setting `SSH_KNOWN_HOSTS`).
- The git informer keeps the last-seen commit baseline in memory. After API server restart/leader failover, each trigger is re-baselined to the current HEAD and commits pushed while the informer was down are not replayed.

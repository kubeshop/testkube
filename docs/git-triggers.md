# Git Event Triggers

Git Event Triggers let you run Test Workflows when files in a Git repository change. They are useful when your test inputs live in Git and you want Testkube to react automatically without depending on an external CI system.

Typical use cases include:

- running regression or smoke workflows when application code changes in a specific branch
- validating API, contract, or configuration changes only when selected folders are updated
- re-running workflows when shared test data, manifests, or reusable test assets are updated

This can be a good alternative to GitHub Actions when you want both the trigger and the execution lifecycle to stay inside Testkube.

## How Git Event Triggers work

For each trigger, Testkube polls the configured repository on a configurable interval (1 minute by default). On each poll, it resolves all matching refs from `branches` / `tags`, then removes refs matching `branchesIgnore` / `tagsIgnore` and applies `paths` / `pathsIgnore`. It compares the current ref commit(s) with the last cached commit(s) for that trigger, and when a new matching change is detected it emits a synthetic git content event and runs the selected workflow.

This guide shows how to configure git-based content triggers using the `ContentGit` fields.

## TestTrigger: watch pull request activity (`git-pull-request`)

`git-pull-request` uses the GitHub API (GitHub.com and GitHub Enterprise Server (GHES)) to poll pull requests for the configured repository. Testkube tracks PR lifecycle and head commit changes per trigger, and emits an event when a supported PR type (`opened`, `synchronize`, `reopened`, `closed`) matches the configured filters (`pullRequest.types`, `pullRequest.branches`, `pullRequest.branchesIgnore`, `paths`, `pathsIgnore`).

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: trigger-pr-main
  namespace: testkube
spec:
  resource: content
  event: git-pull-request
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      tokenFrom:
        secretKeyRef:
          name: gh-token
          key: token
      pullRequest:
        types:
          - opened
          - synchronize
          - reopened
        branches:
          - main
          - release/*
        branchesIgnore:
          - release/legacy-*
      paths:
        - pkg/**
      pathsIgnore:
        - "**/*_test.go"
  action: run
  execution: testworkflow
  testSelector:
    name: my-workflow
    namespace: testkube
```

## TestTrigger: watch any change in branches (`git-push`)

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: trigger-on-main-changes
  namespace: testkube
spec:
  resource: content
  event: git-push
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      branches:
        - main
        - release/*
  action: run
  execution: testworkflow
  testSelector:
    name: my-workflow
    namespace: testkube
```

## TestTrigger: watch tag updates (`git-tag-push`)

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: trigger-release-tags
  namespace: testkube
spec:
  resource: content
  event: git-tag-push
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      tags:
        - v*
      tagsIgnore:
        - v*-rc*
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
  event: git-push
  contentSelector:
    git:
      uri: https://github.com/kubeshop/testkube
      branches:
        - main
      branchesIgnore:
        - main-hotfix/*
      authType: basic
      tokenFrom:
        secretKeyRef:
          name: git-creds
          key: token
      paths:
        - cmd/api-server/**
        - pkg/triggers/**
      pathsIgnore:
        - "**/*_test.go"
  action: run
  execution: testworkflow
  testSelector:
    name: my-workflow
    namespace: testkube
```

## Notes

- Use `event: git-push` for branch refs and `event: git-tag-push` for tag refs.
- Use `event: git-pull-request` to watch pull request activity from GitHub API polling.
- `branches` (example: `["main", "release/*"]`) supports glob patterns. If empty, all branches are watched.
- `branchesIgnore` (example: `["main-hotfix/*", "legacy/*"]`) takes precedence over `branches`.
- `tags` (example: `["v*", "release-*"]`) supports glob patterns for tag refs.
- `tagsIgnore` (example: `["v*-rc*", "v0.*"]`) takes precedence over `tags`.
- `pullRequest.types` (example: `["opened", "synchronize", "reopened"]`) filters PR activity types. If empty, all supported types are watched.
  - Supported values: `opened` (PR created), `synchronize` (new commits pushed to PR head), `reopened` (closed PR reopened), `closed` (PR closed or merged).
- `pullRequest.branches` (example: `["main", "release/*"]`) filters PR base branches and supports glob patterns.
- `pullRequest.branchesIgnore` (example: `["release/legacy-*"]`) excludes base branches and takes precedence over `pullRequest.branches`.
- `paths` (example: `["src/**", "charts/**"]`) is an include filter and supports glob patterns (`/**` matches all descendants).
- `pathsIgnore` (example: `["**/*.md", "docs/**"]`) excludes matching paths and takes precedence over `paths`.
- The polling interval is configurable; by default Testkube checks the repository every 1 minute.
- Testkube caches the last-seen commit per matching ref to detect new changes between polling cycles.
- For `git-pull-request`, Testkube also exports PR metadata variables:
  - `TESTKUBE_GIT_PR_NUMBER`
  - `TESTKUBE_GIT_PR_ACTION`
  - `TESTKUBE_GIT_PR_BASE_REF`
  - `TESTKUBE_GIT_PR_HEAD_REF`
  - `TESTKUBE_GIT_PR_HEAD_SHA`
  - `TESTKUBE_GIT_PR_URL`
  - `TESTKUBE_GIT_PR_TITLE`
  - `TESTKUBE_GIT_PR_AUTHOR`
- After API server restart or leader failover, each trigger is re-baselined to the current refs and commits pushed while the informer was down are not replayed.
- Prefer `tokenFrom` / `sshKeyFrom` (and `usernameFrom`) over inline plain-text fields.
- SSH auth requires host key verification via `known_hosts` (for example by mounting a known_hosts file and setting `SSH_KNOWN_HOSTS`).

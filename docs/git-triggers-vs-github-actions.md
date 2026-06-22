# Testkube Git Triggers: A Vendor-Agnostic Alternative to GitHub Actions for Test Automation

## Introduction

Modern software teams rely on CI/CD pipelines to run tests automatically when code changes. **GitHub Actions** has become a popular choice, but it tightly couples your test automation to a single Git vendor, requires outbound webhook connectivity, and forces teams to manage YAML-based pipeline definitions outside their testing infrastructure.

**Testkube Git Triggers** offer a fundamentally different approach. By polling Git repositories directly from inside your Kubernetes cluster, Testkube enables the same event-driven test automation — **without webhooks, without vendor lock-in, and without internet access**.

This document compares Testkube Git Triggers with GitHub Actions across key scenarios, demonstrates how `git-push`, `git-tag-push`, and `git-pull-request` events work with any Git provider, and shows why Testkube is the superior choice for teams running in air-gapped, multi-cloud, or hybrid environments.

---

## How Testkube Git Triggers Work

Testkube Git Triggers monitor Git repositories using a **polling-based architecture**. A component called the **Git Informer** runs inside your Kubernetes cluster and periodically checks configured repositories for new commits, tags, and pull requests.

### Core Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                             │
│  ┌──────────────┐     ┌──────────────┐    ┌──────────────┐  │
│  │ Git Informer │────▶│   Trigger    │───▶│ TestWorkflow │  │
│  │  (Polling)   │     │   Matcher    │    │  Execution   │  │
│  └──────┬───────┘     └──────────────┘    └──────────────┘  │
│         │                                                    │
│         │  Polls every 1 min (configurable)                  │
└─────────┼───────────────────────────────────────────────────┘
          │
          ▼
   ┌──────────────┐
   │ Any Git Repo │  (GitHub, GitLab, Bitbucket, Gitea,
   │   (Remote)   │   self-hosted, air-gapped)
   └──────────────┘
```

### Supported Events

| Event            | Description                              | Use Case                                    |
|------------------|------------------------------------------|---------------------------------------------|
| `git-push`       | New commits pushed to a branch           | Run tests on every code change              |
| `git-tag-push`   | New tag pushed to the repository         | Run release validation and smoke tests      |
| `git-pull-request` | Pull request opened, updated, or closed | Run PR validation checks (GitHub repos)     |

### Key Differentiators

- **No webhooks required** — Testkube polls repositories, eliminating the need for inbound network access
- **Pure Go implementation** — Uses the `go-git` library natively; no external `git` binary needed
- **Leader-elected** — In multi-replica deployments, only one instance polls to avoid duplicate events
- **Rich metadata injection** — Git context (commit SHA, branch, tag, PR details) is automatically available in your test workflows

---

## Side-by-Side Comparison: Testkube Git Triggers vs. GitHub Actions

### At a Glance

| Capability                        | GitHub Actions                          | Testkube Git Triggers                        |
|-----------------------------------|-----------------------------------------|----------------------------------------------|
| **Trigger Mechanism**             | Webhook-based                           | Polling-based (no inbound traffic)           |
| **Git Vendor Support**            | GitHub only                             | Any Git provider (GitHub, GitLab, Bitbucket, Gitea, self-hosted) |
| **Air-Gapped Support**            | ❌ Requires internet connectivity       | ✅ Works fully offline with internal repos   |
| **Test Execution Environment**    | GitHub-hosted or self-hosted runners    | Your own Kubernetes cluster                  |
| **Secrets Management**            | GitHub Secrets (per-repo/org)           | Kubernetes Secrets & ConfigMaps              |
| **Branch/Tag Filtering**          | Glob patterns in YAML                   | Glob patterns in CRD spec                    |
| **Path Filtering**                | `paths` / `paths-ignore` in YAML        | `paths` / `pathsIgnore` in CRD spec         |
| **Concurrency Control**           | `concurrency` groups                    | `allow` / `forbid` / `replace` policies     |
| **Test Orchestration**            | Step-based YAML workflows               | Testkube TestWorkflows (purpose-built)       |
| **Artifact Management**           | GitHub Artifacts API                    | Built-in artifact storage (S3, MinIO, etc.)  |
| **Dashboard & Reporting**         | GitHub Actions tab                      | Testkube Dashboard with deep test analytics  |
| **Multi-Cluster**                 | ❌ Single runner scope                  | ✅ Centralized control across clusters       |
| **Cost Model**                    | Per-minute billing for hosted runners   | Run on your existing infrastructure          |

---

## Scenario 1: Run Tests on Every Push to `main`

### GitHub Actions

```yaml
# .github/workflows/test-on-push.yml
name: Run Tests on Push
on:
  push:
    branches:
      - main
    paths:
      - 'src/**'
      - 'tests/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: |
          npm install
          npm test
```

**Limitations:**
- Tied to GitHub — moving to GitLab or Bitbucket requires a complete rewrite
- Tests run on ephemeral GitHub runners, not in your actual deployment environment
- No access to in-cluster services, databases, or APIs without complex tunneling

### Testkube Git Triggers

```yaml
# TestTrigger CRD
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: test-on-main-push
  namespace: testkube
spec:
  resource: content
  event: git-push
  contentSelector:
    git:
      uri: https://github.com/your-org/your-repo.git
      branches:
        - main
      paths:
        - "src/**"
        - "tests/**"
      tokenFrom:
        secretKeyRef:
          name: git-credentials
          key: token
  action: run
  execution: testworkflow
  testSelector:
    name: integration-tests
    namespace: testkube
  concurrencyPolicy: replace
```

```yaml
# TestWorkflow
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: integration-tests
  namespace: testkube
spec:
  content:
    git:
      uri: https://github.com/your-org/your-repo.git
      revision: "{{ config.TESTKUBE_GIT_COMMIT }}"
  steps:
    - name: Run Tests
      run:
        image: node:20
        shell: |
          cd /data/repo
          npm install
          npm test
```

**Advantages:**
- Tests run **inside your Kubernetes cluster** with access to real services, databases, and APIs
- The same trigger works with **any Git provider** — just change the `uri`
- Git metadata (`TESTKUBE_GIT_COMMIT`, `TESTKUBE_GIT_BRANCH`) is automatically injected
- `concurrencyPolicy: replace` ensures only the latest push is tested

---

## Scenario 2: Release Validation on Tag Push

### GitHub Actions

```yaml
# .github/workflows/release-tests.yml
name: Release Validation
on:
  push:
    tags:
      - 'v*'

jobs:
  smoke-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to staging
        run: ./deploy.sh staging
      - name: Run smoke tests
        run: ./run-smoke-tests.sh
      - name: Run performance tests
        run: ./run-perf-tests.sh
```

**Limitations:**
- Cannot easily deploy to a real staging cluster from GitHub runners
- Performance tests on shared GitHub infrastructure produce unreliable results
- No native integration with Kubernetes deployments

### Testkube Git Triggers

```yaml
# TestTrigger CRD
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: release-validation
  namespace: testkube
spec:
  resource: content
  event: git-tag-push
  contentSelector:
    git:
      uri: https://github.com/your-org/your-repo.git
      tags:
        - "v*"
      tagsIgnore:
        - "v*-rc*"
        - "v*-beta*"
      tokenFrom:
        secretKeyRef:
          name: git-credentials
          key: token
  action: run
  execution: testworkflow
  testSelector:
    name: release-smoke-tests
    namespace: testkube
```

```yaml
# TestWorkflow for release validation
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: release-smoke-tests
  namespace: testkube
spec:
  steps:
    - name: Deploy to Staging
      run:
        image: bitnami/kubectl:latest
        shell: |
          kubectl set image deployment/myapp \
            myapp=myregistry/myapp:{{ config.TESTKUBE_GIT_TAG }}
          kubectl rollout status deployment/myapp --timeout=120s

    - name: Smoke Tests
      run:
        image: grafana/k6:latest
        shell: |
          k6 run /data/tests/smoke.js

    - name: Performance Tests
      run:
        image: grafana/k6:latest
        shell: |
          k6 run /data/tests/performance.js
```

**Advantages:**
- Tag name is available via `TESTKUBE_GIT_TAG` — use it to deploy the exact version
- Smoke and performance tests run **against real infrastructure** in your cluster
- Exclude pre-release tags (`v*-rc*`, `v*-beta*`) with `tagsIgnore` patterns
- Performance test results are consistent — no shared runner variability

---

## Scenario 3: Multi-Service Monorepo Testing

### GitHub Actions

```yaml
# .github/workflows/service-tests.yml
name: Service Tests
on:
  push:
    branches: [main, 'release/*']

jobs:
  detect-changes:
    runs-on: ubuntu-latest
    outputs:
      api: ${{ steps.changes.outputs.api }}
      web: ${{ steps.changes.outputs.web }}
    steps:
      - uses: dorny/paths-filter@v3
        id: changes
        with:
          filters: |
            api:
              - 'services/api/**'
            web:
              - 'services/web/**'

  test-api:
    needs: detect-changes
    if: ${{ needs.detect-changes.outputs.api == 'true' }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cd services/api && go test ./...

  test-web:
    needs: detect-changes
    if: ${{ needs.detect-changes.outputs.web == 'true' }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: cd services/web && npm test
```

**Limitations:**
- Complex change detection logic with third-party actions
- Conditional job execution adds pipeline complexity
- Each service test runs in isolation, unable to test cross-service interactions

### Testkube Git Triggers

```yaml
# One trigger per service — path filtering is built in
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: api-service-trigger
  namespace: testkube
spec:
  resource: content
  event: git-push
  contentSelector:
    git:
      uri: https://github.com/your-org/monorepo.git
      branches:
        - main
        - "release/*"
      paths:
        - "services/api/**"
      pathsIgnore:
        - "services/api/**/*.md"
        - "services/api/docs/**"
  action: run
  execution: testworkflow
  testSelector:
    name: api-service-tests
    namespace: testkube
---
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: web-service-trigger
  namespace: testkube
spec:
  resource: content
  event: git-push
  contentSelector:
    git:
      uri: https://github.com/your-org/monorepo.git
      branches:
        - main
        - "release/*"
      paths:
        - "services/web/**"
      pathsIgnore:
        - "services/web/**/*.md"
  action: run
  execution: testworkflow
  testSelector:
    name: web-service-tests
    namespace: testkube
```

**Advantages:**
- **Native path filtering** — no third-party actions or change detection hacks
- Each trigger is independent and declarative — easy to understand and maintain
- Services can be tested in isolation **or** together with cross-service integration tests
- `pathsIgnore` ensures documentation-only changes don't trigger test runs

---

## Scenario 4: Air-Gapped and Self-Hosted Git Environments

This is where Testkube Git Triggers truly shine — and where GitHub Actions simply cannot operate.

### The Problem with GitHub Actions in Restricted Environments

GitHub Actions fundamentally requires:
1. **Outbound internet access** to reach GitHub's API and download actions
2. **Webhook delivery** from GitHub to trigger workflows
3. **GitHub-hosted runners** or self-hosted runners with GitHub connectivity
4. **GitHub as your Git provider** — it does not work with GitLab, Bitbucket, or self-hosted Git

For organizations in defense, healthcare, finance, or government sectors, these requirements are often non-starters.

### Testkube Git Triggers: Built for Air-Gapped Environments

```yaml
# Works with any internal Git server — no internet required
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: air-gapped-trigger
  namespace: testkube
spec:
  resource: content
  event: git-push
  contentSelector:
    git:
      uri: git@git.internal.corp:team/secure-app.git
      branches:
        - main
        - "release/*"
      authType: ssh
      sshKeyFrom:
        secretKeyRef:
          name: internal-git-ssh
          key: private-key
  action: run
  execution: testworkflow
  testSelector:
    name: security-compliance-tests
    namespace: testkube
```

### Why It Works Without Internet

| Requirement         | GitHub Actions                            | Testkube Git Triggers                       |
|---------------------|-------------------------------------------|---------------------------------------------|
| **Network Model**   | Requires outbound internet + webhooks     | Only needs network path to Git server       |
| **Git Protocol**    | HTTPS to github.com                       | HTTPS, SSH, or Git protocol to any server   |
| **Authentication**  | GitHub OAuth / PAT                        | SSH keys, basic auth, or tokens from K8s Secrets |
| **Runtime**         | GitHub cloud or runner with GitHub access | Kubernetes pods — fully self-contained      |
| **Dependencies**    | Downloads actions from GitHub Marketplace | Container images from your private registry |
| **Git Library**     | External `git` binary                     | Pure Go `go-git` library — no binary needed |

### Supported Git Providers

Testkube Git Triggers work identically across all Git providers:

| Provider                | URI Example                                      | Auth Method         |
|-------------------------|--------------------------------------------------|---------------------|
| **GitHub**              | `https://github.com/org/repo.git`                | Token, SSH          |
| **GitHub Enterprise**   | `https://github.corp.com/org/repo.git`           | Token, SSH          |
| **GitLab**              | `https://gitlab.com/org/repo.git`                | Token, SSH          |
| **GitLab Self-Managed** | `https://gitlab.internal.com/org/repo.git`       | Token, SSH, Basic   |
| **Bitbucket**           | `https://bitbucket.org/org/repo.git`             | Token, SSH          |
| **Bitbucket Server**    | `https://bitbucket.corp.com/scm/proj/repo.git`  | Token, SSH, Basic   |
| **Azure DevOps**        | `https://dev.azure.com/org/proj/_git/repo`       | Token, SSH          |
| **Gitea**               | `https://gitea.internal.com/org/repo.git`        | Token, SSH, Basic   |
| **AWS CodeCommit**      | `https://git-codecommit.region.amazonaws.com/repo` | Basic (IAM)       |
| **Any Git server**      | `git@git.internal:repo.git`                      | SSH                 |

The same trigger YAML works with every provider — only the `uri` and credentials change.

---

## Scenario 5: Tag-Based Release Pipeline Across Git Vendors

A common CI/CD pattern is triggering release workflows when a version tag is pushed. With GitHub Actions, this only works on GitHub. With Testkube, it works everywhere.

### GitHub Actions (GitHub Only)

```yaml
on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'
```

### Testkube Git Triggers (Any Git Vendor)

```yaml
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: release-pipeline
  namespace: testkube
spec:
  resource: content
  event: git-tag-push
  contentSelector:
    git:
      # Works with ANY Git server
      uri: https://gitlab.internal.com/platform/core-service.git
      tags:
        - "v*"
      tagsIgnore:
        - "v*-dev"
        - "v*-snapshot"
      tokenFrom:
        secretKeyRef:
          name: gitlab-credentials
          key: token
  action: run
  execution: testworkflow
  testSelector:
    nameRegex: "release-.*"
    namespace: testkube
```

When a tag like `v2.5.0` is pushed to **any** Git server, Testkube:

1. **Detects the tag** via polling (no webhook needed)
2. **Injects metadata** — `TESTKUBE_GIT_TAG=v2.5.0`, `TESTKUBE_GIT_COMMIT=abc123`, `TESTKUBE_GIT_REF=refs/tags/v2.5.0`
3. **Runs all matching TestWorkflows** — e.g., `release-smoke-tests`, `release-integration-tests`, `release-security-scan`
4. **Reports results** in the Testkube Dashboard with full traceability

---

## Scenario 6: Pull Request Validation — Keep GitHub, Run Tests in Your Own Environment

Many teams are deeply integrated with GitHub — their workflows, code reviews, branch protection rules, and developer tooling all revolve around GitHub Pull Requests. Moving away from GitHub entirely is impractical or undesirable. However, running test workloads on GitHub-hosted runners means tests execute **outside your real infrastructure**, without access to internal services, databases, or production-like environments.

**Testkube's `git-pull-request` event** bridges this gap: you keep GitHub as your source-of-truth for code and collaboration, while Testkube runs your PR validation tests **inside your own Kubernetes cluster** — against real services, with real data, in a real network topology.

### The Problem: GitHub Actions PR Tests Run in Isolation

```yaml
# .github/workflows/pr-tests.yml
name: PR Validation
on:
  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Start dependencies
        run: docker-compose up -d  # Simulated services, not real ones
      - name: Run integration tests
        run: |
          npm install
          npm run test:integration
      - name: Run E2E tests
        run: npm run test:e2e  # Against mocked APIs, not real cluster
```

**Limitations of this approach:**
- Tests run against **mocked or containerized services**, not your real infrastructure
- No access to internal APIs, databases, or message queues behind your firewall
- Docker-compose "mini environments" **diverge from production** — passing tests don't guarantee production readiness
- Self-hosted runners require exposing your infrastructure to GitHub's webhook delivery network
- GitHub-hosted runners add latency (provisioning, dependency installation) on every PR update
- Cannot test Kubernetes-specific behaviors (network policies, service mesh, RBAC, resource limits)

### Testkube `git-pull-request` Event: GitHub for Code, Your Cluster for Tests

```yaml
# TestTrigger CRD — reacts to GitHub Pull Requests
apiVersion: tests.testkube.io/v1
kind: TestTrigger
metadata:
  name: pr-validation
  namespace: testkube
spec:
  resource: content
  event: git-pull-request
  contentSelector:
    git:
      uri: https://github.com/your-org/your-repo.git
      pullRequest:
        branches:
          - main            # Only PRs targeting main
      paths:
        - "src/**"
        - "tests/**"
        - "api/**"
      pathsIgnore:
        - "**/*.md"
        - "docs/**"
      tokenFrom:
        secretKeyRef:
          name: github-token
          key: token
  action: run
  execution: testworkflow
  testSelector:
    name: pr-integration-tests
    namespace: testkube
  concurrencyPolicy: replace   # Only test the latest PR commit
```

```yaml
# TestWorkflow — runs PR validation inside your cluster
apiVersion: testworkflows.testkube.io/v1
kind: TestWorkflow
metadata:
  name: pr-integration-tests
  namespace: testkube
spec:
  content:
    git:
      uri: https://github.com/your-org/your-repo.git
      revision: "{{ config.TESTKUBE_GIT_PR_HEAD_SHA }}"
      tokenFrom:
        secretKeyRef:
          name: github-token
          key: token
  steps:
    - name: Integration Tests Against Real Services
      run:
        image: node:20
        env:
          - name: DATABASE_URL
            valueFrom:
              secretKeyRef:
                name: test-db-credentials
                key: url
          - name: API_GATEWAY_URL
            value: "http://api-gateway.staging.svc.cluster.local:8080"
        shell: |
          echo "Testing PR #{{ config.TESTKUBE_GIT_PR_NUMBER }}: {{ config.TESTKUBE_GIT_PR_TITLE }}"
          echo "Branch: {{ config.TESTKUBE_GIT_PR_HEAD_REF }} → {{ config.TESTKUBE_GIT_PR_BASE_REF }}"
          cd /data/repo
          npm install
          npm run test:integration   # Tests against REAL database
          npm run test:e2e           # Tests against REAL API gateway

    - name: Kubernetes-Specific Validation
      run:
        image: bitnami/kubectl:latest
        shell: |
          # Deploy the PR branch to a preview namespace
          kubectl create namespace preview-pr-{{ config.TESTKUBE_GIT_PR_NUMBER }} --dry-run=client -o yaml | kubectl apply -f -
          kubectl apply -f k8s/manifests/ -n preview-pr-{{ config.TESTKUBE_GIT_PR_NUMBER }}
          kubectl rollout status deployment/app -n preview-pr-{{ config.TESTKUBE_GIT_PR_NUMBER }} --timeout=120s

    - name: Performance Baseline Check
      run:
        image: grafana/k6:latest
        shell: |
          # Run performance tests against the preview deployment
          k6 run --out json=/data/artifacts/perf-results.json \
            /data/repo/tests/performance/baseline.js
```

### How It Works

1. **Developer opens or updates a PR on GitHub** — normal GitHub workflow, no changes needed
2. **Testkube's Git Informer detects the PR event** via polling (no webhook configuration required)
3. **Trigger matches** — the PR targets `main` and changes files in `src/**`, `tests/**`, or `api/**`
4. **TestWorkflow executes inside your cluster** — with full access to internal services, databases, and APIs
5. **Rich PR metadata is injected** — PR number, title, author, head/base refs, head SHA, and PR URL are all available as `config.*` variables
6. **Results appear in the Testkube Dashboard** — with full test logs, artifacts, and traceability back to the PR

### Why This Matters for GitHub-Linked Teams

| Aspect | GitHub Actions Runners | Testkube `git-pull-request` |
|--------|------------------------|------------------------------|
| **Code & Collaboration** | GitHub ✅ | GitHub ✅ (unchanged) |
| **Test Execution** | GitHub cloud / self-hosted runners | Your Kubernetes cluster |
| **Access to Internal Services** | ❌ Requires tunnels/proxies | ✅ Native cluster networking |
| **Access to Real Databases** | ❌ Needs container stubs | ✅ Connect to staging/test DBs directly |
| **Kubernetes-native Testing** | ❌ Limited | ✅ Full kubectl, helm, service mesh access |
| **Network Policy Testing** | ❌ Impossible | ✅ Tests run under real network policies |
| **Performance Test Accuracy** | ❌ Shared runner variability | ✅ Consistent, dedicated infrastructure |
| **Data Compliance** | ⚠️ Data leaves your network | ✅ All data stays within your infrastructure |
| **Webhook Configuration** | Required (repo admin access) | Not required (polling-based) |
| **Branch Protection Integration** | Native check runs | Dashboard results + optional status API |

### The Hybrid Model: Best of Both Worlds

With `git-pull-request`, teams get the **best of both ecosystems**:

- **Keep GitHub for what it does best** — code hosting, pull request reviews, issue tracking, branch protection, team collaboration
- **Use Testkube for what it does best** — running tests inside your actual infrastructure with access to real services, real data, and real network conditions

This means:
- ✅ Developers continue using their familiar GitHub PR workflow — no behavior change
- ✅ Tests validate against **production-like environments**, not mocked services
- ✅ Sensitive test data (credentials, customer data, internal APIs) **never leaves your network**
- ✅ No webhook endpoints to configure or secure — Testkube polls GitHub via standard Git protocols
- ✅ Works even if your cluster has **no inbound internet access** — only outbound HTTPS to github.com is needed
- ✅ PR metadata (number, title, author, branch names) flows into tests automatically for reporting and traceability

### Example: Complete PR Workflow with Status Reporting

For teams that want PR check status reported back to GitHub, Testkube can be paired with a simple status update step:

```yaml
steps:
  - name: Run Tests
    run:
      image: node:20
      shell: |
        cd /data/repo && npm test

  - name: Report Status to GitHub
    condition: always
    run:
      image: curlimages/curl:latest
      env:
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-token
              key: token
      shell: |
        STATUS=$([ "{{ status }}" = "passed" ] && echo "success" || echo "failure")
        curl -X POST \
          -H "Authorization: token $GITHUB_TOKEN" \
          -H "Accept: application/vnd.github.v3+json" \
          "https://api.github.com/repos/your-org/your-repo/statuses/{{ config.TESTKUBE_GIT_PR_HEAD_SHA }}" \
          -d "{\"state\":\"$STATUS\",\"target_url\":\"https://app.testkube.io/executions/{{ execution.id }}\",\"description\":\"Testkube PR Validation\",\"context\":\"testkube/pr-tests\"}"
```

This posts a commit status back to GitHub, so the PR shows a green/red check — maintaining full visibility in the GitHub UI while tests execute in your environment.

---

## Git Metadata: Context Available in Your Tests

Every triggered TestWorkflow receives rich Git context as environment variables, enabling dynamic test behavior:

### Standard Git Metadata

| Variable                  | Example Value                   | Available On           |
|---------------------------|---------------------------------|------------------------|
| `TESTKUBE_GIT_COMMIT`     | `a1b2c3d4e5f6`                 | All git events         |
| `TESTKUBE_GIT_REF`        | `refs/heads/main`              | All git events         |
| `TESTKUBE_GIT_BRANCH`     | `main`                         | `git-push`             |
| `TESTKUBE_GIT_TAG`        | `v1.2.3`                       | `git-tag-push`         |

### Pull Request Metadata (GitHub Repos)

| Variable                    | Example Value                                    |
|-----------------------------|--------------------------------------------------|
| `TESTKUBE_GIT_PR_NUMBER`    | `142`                                            |
| `TESTKUBE_GIT_PR_ACTION`    | `opened`                                         |
| `TESTKUBE_GIT_PR_BASE_REF`  | `main`                                           |
| `TESTKUBE_GIT_PR_HEAD_REF`  | `feature/new-api`                                |
| `TESTKUBE_GIT_PR_HEAD_SHA`  | `f7e8d9c0`                                       |
| `TESTKUBE_GIT_PR_URL`       | `https://github.com/org/repo/pull/142`           |
| `TESTKUBE_GIT_PR_TITLE`     | `Add new API endpoint`                           |
| `TESTKUBE_GIT_PR_AUTHOR`    | `jsmith`                                         |

### Using Metadata in TestWorkflows

```yaml
steps:
  - name: Test against correct commit
    run:
      image: golang:1.22
      shell: |
        echo "Testing commit: {{ config.TESTKUBE_GIT_COMMIT }}"
        echo "Branch: {{ config.TESTKUBE_GIT_BRANCH }}"
        echo "Tag: {{ config.TESTKUBE_GIT_TAG }}"
        git checkout {{ config.TESTKUBE_GIT_COMMIT }}
        go test ./...
```

---

## Advanced Capabilities

### Concurrency Policies

Control how triggers behave when multiple pushes arrive in quick succession:

| Policy    | Behavior                                                      |
|-----------|---------------------------------------------------------------|
| `allow`   | Run all triggered executions concurrently                     |
| `forbid`  | Skip execution if a previous one is still running             |
| `replace` | Abort the running execution and start a new one               |

### Branch and Tag Glob Patterns

```yaml
contentSelector:
  git:
    branches:
      - main
      - "release/*"          # Matches release/1.0, release/2.0, etc.
      - "feature/**"         # Matches feature/auth, feature/api/v2, etc.
    branchesIgnore:
      - "dependabot/**"      # Ignore automated dependency updates
    tags:
      - "v[0-9]*"            # Matches v1.0.0, v2.1.3, etc.
    tagsIgnore:
      - "v*-rc*"             # Ignore release candidates
```

### Path-Based Filtering

Only trigger when relevant files change — avoid unnecessary test runs:

```yaml
contentSelector:
  git:
    paths:
      - "src/**"
      - "tests/**"
      - "go.mod"
      - "go.sum"
    pathsIgnore:
      - "**/*.md"
      - "docs/**"
      - ".github/**"
      - "LICENSE"
```

### Configurable Polling Behavior

Fine-tune the Git Informer for your environment:

| Environment Variable                                  | Default  | Description                              |
|-------------------------------------------------------|----------|------------------------------------------|
| `TEST_TRIGGER_GIT_INFORMER_RECONCILE_INTERVAL`        | `1m`     | How often to check for new commits       |
| `TEST_TRIGGER_GIT_INFORMER_REPO_DEPTH`                | `500`    | Clone depth for repositories             |
| `TEST_TRIGGER_GIT_INFORMER_MAX_COMMITS_SCAN`          | `500`    | Max commits to scan per reconciliation   |
| `TEST_TRIGGER_GIT_INFORMER_LIST_TIMEOUT`              | `15`     | Timeout in seconds for git ls-remote operations |
| `TEST_TRIGGER_GIT_INFORMER_PULL_RETRIES`              | `2`      | Number of retries on git pull failures   |
| `TEST_TRIGGER_GIT_INFORMER_PULL_RETRY_DELAY`          | `2s`     | Delay between pull retries               |

---

## Migration Guide: From GitHub Actions to Testkube Git Triggers

### Step 1: Identify Your Triggers

Map your GitHub Actions `on:` blocks to Testkube events:

| GitHub Actions `on:`    | Testkube Event       |
|-------------------------|----------------------|
| `push.branches`         | `git-push`           |
| `push.tags`             | `git-tag-push`       |
| `pull_request`          | `git-pull-request`   |

### Step 2: Convert Path Filters

| GitHub Actions                     | Testkube                          |
|------------------------------------|-----------------------------------|
| `paths: ['src/**']`               | `paths: ["src/**"]`              |
| `paths-ignore: ['docs/**']`       | `pathsIgnore: ["docs/**"]`       |

### Step 3: Move Secrets to Kubernetes

```bash
# GitHub Secret → Kubernetes Secret
kubectl create secret generic my-test-secrets \
  --from-literal=API_KEY=your-api-key \
  --from-literal=DB_PASSWORD=your-db-pass \
  -n testkube
```

### Step 4: Create TestWorkflows

Convert your GitHub Actions job steps into Testkube TestWorkflow steps. Each `run:` block maps to a TestWorkflow step with the appropriate container image.

### Step 5: Deploy TestTrigger CRDs

Apply your trigger definitions to your Kubernetes cluster:

```bash
kubectl apply -f triggers/ -n testkube
```

---

## Why Teams Choose Testkube Git Triggers

### 🔒 Security & Compliance
- No inbound webhook endpoints to secure
- Tests run inside your own infrastructure — data never leaves your network
- Kubernetes RBAC and network policies apply naturally
- Full audit trail in the Testkube Dashboard

### 🌐 Vendor Independence
- Works with **every Git provider** — switch vendors without rewriting pipelines
- No marketplace dependency — use any container image as a test runner
- Portable across clouds, on-premises, and hybrid environments

### 🏔️ Air-Gapped Ready
- Polling-based architecture requires only outbound access to your Git server
- Pure Go Git implementation — no external binaries
- All components run as Kubernetes workloads using your private container registry

### ⚡ Purpose-Built for Testing
- Native support for test frameworks (k6, Playwright, Cypress, JMeter, etc.)
- Built-in artifact collection and test analytics
- Parallel test execution across Kubernetes nodes
- Flaky test detection and historical trend analysis

### 💰 Cost Efficiency
- No per-minute runner billing — use your existing Kubernetes capacity
- Shared cluster resources across all test workloads
- Scale test parallelism with Kubernetes auto-scaling

---

## Summary

Testkube Git Triggers provide a **vendor-agnostic, air-gap-compatible, Kubernetes-native** alternative to GitHub Actions for test automation. By using a polling-based architecture instead of webhooks, Testkube eliminates the dependency on any single Git vendor and enables test automation in environments where GitHub Actions simply cannot operate.

For teams deeply invested in GitHub, the `git-pull-request` event offers the best of both worlds: continue using GitHub for code hosting, pull requests, and collaboration while running your test workloads inside your own Kubernetes cluster — with access to real services, real databases, and production-like network conditions.

Whether you're running on GitHub, GitLab, Bitbucket, Azure DevOps, or a self-hosted Gitea instance behind a firewall — Testkube Git Triggers give you the same powerful, event-driven test automation with the added benefits of running tests inside your actual infrastructure.

| | GitHub Actions | Testkube Git Triggers |
|---|---|---|
| **Best For** | Simple CI/CD on GitHub repos | Enterprise test automation across any Git vendor |
| **Environment** | Cloud-hosted runners | Your Kubernetes cluster |
| **Network** | Requires internet | Works fully air-gapped |
| **Vendor Lock-in** | GitHub only | Any Git provider |
| **Test Context** | Isolated runners | Real infrastructure with services and APIs |

**Ready to get started?** Visit [testkube.io](https://testkube.io) to learn more, or check out the [Testkube documentation](https://docs.testkube.io) for detailed setup instructions.

# testkube-runner Helm unit tests

Suites for [`helm-unittest`](https://github.com/helm-unittest/helm-unittest) covering every template in the chart.

## Run

```bash
helm plugin install https://github.com/helm-unittest/helm-unittest.git  # once
helm unittest k8s/helm/testkube-runner
```

Expected: `Test Suites: 8 passed, 8 total` / `Tests: 75 passed`.

## Suites

| Suite file | Template | Tests | Coverage |
|---|---|---|---|
| `service_test.yaml` | `service.yaml` | 11 | port 8088, ClusterIP, selector, labels/annotations propagation |
| `servicemonitor_test.yaml` | `servicemonitor.yaml` | 13 | gating by `prometheus.enabled`, targetPort sync with Service, interval/monitoringLabels/matchLabels/sampleLimit |
| `serviceaccount_test.yaml` | `serviceaccount.yaml` | 8 | auto-create branches (`pod.serviceAccount.autoCreate`, `execution.default.serviceAccount.autoCreate`), name/namespace overrides, annotations |
| `poddisruptionbudget_test.yaml` | `poddisruptionbudget.yaml` | 7 | local vs `global.podDisruptionBudget` fallback, `minAvailable`/`maxUnavailable`, selector, K8s ≥1.21 apiVersion |
| `role_test.yaml` | `role.yaml` | 10 | the 4 base Roles + watchers ClusterRole/Role per `watchAllNamespaces`; gating via `listener.enabled` / `gitops.enabled` |
| `rolebinding_test.yaml` | `rolebinding.yaml` | 10 | RoleBinding subjects + roleRef wiring (`agent-sa` → exec-role for jobs, `exec-sa` → exec-role), namespace propagation, watchers ClusterRoleBinding |
| `deployment_test.yaml` | `deployment.yaml` | 15 | port 8088, ServiceAccount, runner credentials (inline & via secretRef), TLS, listener/gitops toggles, registration token, image references |
| `crds_test.yaml` | `crds.yaml` | 1 | CRDs are rendered when `gitops.enabled && gitops.installCRD` |

CI does not run these yet — they are local-dev guardrails. Wiring into a workflow is a separate follow-up.

## Conventions

- Assertion style: explicit `equal` / `contains` / `matchRegex` / `hasDocuments` / `lengthEqual`. No snapshot tests.
- For multi-document templates (`role.yaml`, `rolebinding.yaml`, `serviceaccount.yaml`), use `documentSelector` on `metadata.name` to target individual resources — document ordering between helm and helm-unittest can disagree.
- Use `templates/<file>.yaml` (full path) when a template `include`s subchart helpers, e.g. `crds.yaml`. The short form (`<file>.yaml`) works for templates with no subchart includes.

## Adding a test

```yaml
suite: <description>
templates:
  - <template>.yaml          # or templates/<template>.yaml if it includes from a subchart
release:
  name: tk
  namespace: testkube-dev
tests:
  - it: <what this verifies>
    set:                     # optional overrides
      some.value: foo
    documentSelector:        # optional, for multi-doc templates
      path: metadata.name
      value: my-resource
    asserts:
      - equal:
          path: spec.foo
          value: bar
```

Run a single suite during iteration:

```bash
helm unittest k8s/helm/testkube-runner -f tests/<suite>.yaml
```

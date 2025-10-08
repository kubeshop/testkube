# testkube-operator

![Version: 1.14.0](https://img.shields.io/badge/Version-1.14.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.14.0](https://img.shields.io/badge/AppVersion-1.14.0-informational?style=flat-square)

A Helm chart for the testkube-operator (installs needed CRDs only for now)

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| file://../global | global | 0.1.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| apiFullname | string | `"testkube-api-server"` |  |
| apiPort | int | `8088` |  |
| agentCronJobs | bool | `true` |  |
| useArgoCDSync| bool | `false` |  |
| extraEnvVars | list | `[]` |  |
| fullnameOverride | string | `""` |  |
| global.annotations | object | `{}` |  |
| global.imagePullSecrets | list | `[]` |  |
| global.imageRegistry | string | `""` |  |
| global.labels | object | `{}` |  |
| healthcheckPort | int | `8081` |  |
| image.digest | string | `""` |  |
| image.pullPolicy | string | `""` |  |
| image.pullSecrets | list | `[]` |  |
| image.registry | string | `"docker.io"` |  |
| image.repository | string | `"kubeshop/testkube-operator"` |  |
| args | list | `--logtostderr=true` | |
| installCRD | bool | `true` |  |
| kubeVersion | string | `""` |  |
| livenessProbe.initialDelaySeconds | int | `3` |  |
| livenessProbe.periodSeconds | int | `10` |  |
| metricsServiceName | string | `""` |  |
| nameOverride | string | `""` |  |
| namespace | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| preUpgrade.annotations | object | `{}` |  |
| preUpgrade.enabled | bool | `true` | Upgrade hook is enabled |
| preUpgrade.image | object | `{"pullPolicy":"IfNotPresent","pullSecrets":[],"registry":"docker.io","repository":"bitnami/kubectl","tag":"1.28.2"}` | Specify image parameters |
| preUpgrade.labels | object | `{}` |  |
| preUpgrade.podAnnotations | object | `{}` |  |
| preUpgrade.podSecurityContext | object | `{}` | Upgrade Pod Security Context |
| preUpgrade.resources | object | `{}` | Specify resource limits and requests |
| preUpgrade.securityContext | object | `{}` | Security Context for Upgrade kubectl container |
| preUpgrade.serviceAccount | object | `{"create":true}` | Create SA for upgrade hook |
| preUpgrade.tolerations | list | `[{"effect":"NoSchedule","key":"kubernetes.io/arch","operator":"Equal","value":"arm64"}]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| priorityClassName | string | `""` |  |
| proxy.image.pullPolicy | string | `"IfNotPresent"` |  |
| proxy.image.pullSecrets | list | `[]` |  |
| proxy.image.registry | string | `"gcr.io"` |  |
| proxy.image.repository | string | `"kubebuilder/kube-rbac-proxy"` |  |
| proxy.image.tag | string | `"v0.8.0"` |  |
| proxy.resources | object | `{}` |  |
| purgeExecutions | bool | `false` |  |
| rbac.create | bool | `true` |  |
| readinessProbe.initialDelaySeconds | int | `3` |  |
| readinessProbe.periodSeconds | int | `10` |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext | object | `{}` |  |
| service.annotations | object | `{}` |  |
| service.port | int | `80` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| terminationGracePeriodSeconds | int | `10` |  |
| testConnection.enabled | bool | `false` |  |
| tolerations | list | `[]` |  |
| volumes.secret.defaultMode | int | `420` |  |
| webhook.annotations | object | `{}` |  |
| webhook.certificate.secretName | string | `"webhook-server-cert"` |  |
| webhook.enabled | bool | `true` |  |
| webhook.labels | object | `{}` |  |
| webhook.migrate.backoffLimit | int | `1` |  |
| webhook.migrate.enabled | bool | `true` |  |
| webhook.migrate.image.pullPolicy | string | `"Always"` |  |
| webhook.migrate.image.pullSecrets | list | `[]` |  |
| webhook.migrate.image.registry | string | `"docker.io"` |  |
| webhook.migrate.image.repository | string | `"rancher/kubectl"` |  |
| webhook.migrate.image.tag | string | `"v1.23.7"` |  |
| webhook.migrate.resources | object | `{}` |  |
| webhook.migrate.securityContext | object | `{}` |  |
| webhook.name | string | `"webhook-admission"` |  |
| webhook.patch.annotations | object | `{}` |  |
| webhook.patch.backoffLimit | int | `1` |  |
| webhook.patch.createSecretJob.resources | object | `{}` |  |
| webhook.patch.createSecretJob.securityContext | object | `{}` |  |
| webhook.patch.enabled | bool | `true` |  |
| webhook.patch.image.pullPolicy | string | `"IfNotPresent"` |  |
| webhook.patch.image.pullSecrets | list | `[]` |  |
| webhook.patch.image.registry | string | `"docker.io"` |  |
| webhook.patch.image.repository | string | `"kubeshop/kube-webhook-certgen"` |  |
| webhook.patch.image.tag | string | `"1.0.11"` |  |
| webhook.patch.labels | object | `{}` |  |
| webhook.patch.nodeSelector."kubernetes.io/os" | string | `"linux"` |  |
| webhook.patch.patchWebhookJob.resources | object | `{}` |  |
| webhook.patch.patchWebhookJob.securityContext | object | `{}` |  |
| webhook.patch.podAnnotations | object | `{}` |  |
| webhook.patch.podSecurityContext | object | `{}` |  |
| webhook.patch.serviceAccount.annotations | object | `{}` |  |
| webhook.patch.serviceAccount.name | string | `"testkube-operator-webhook-cert-mgr"` |  |
| webhook.patch.tolerations | list | `[]` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.2](https://github.com/norwoodj/helm-docs/releases/v1.11.2)

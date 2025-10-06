# testkube-api

![Version: 2.0.10](https://img.shields.io/badge/Version-2.0.10-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.0.10](https://img.shields.io/badge/AppVersion-2.0.10-informational?style=flat-square)

A Helm chart for Testkube api

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| file://../global | global | 0.1.2 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| additionalJobVolumeMounts | list | `[]` |  |
| additionalJobVolumes | list | `[]` |  |
| additionalNamespaces | list | `[]` |  |
| additionalVolumeMounts | list | `[]` |  |
| additionalVolumes | list | `[]` |  |
| affinity | object | `{}` |  |
| allowLowSecurityFields |  bool | `false` |  |
| analyticsEnabled | bool | `true` |  |
| autoscaling.annotations | object | `{}` |  |
| autoscaling.enabled | bool | `false` |  |
| autoscaling.labels | object | `{}` |  |
| autoscaling.maxReplicas | int | `100` |  |
| autoscaling.minReplicas | int | `1` |  |
| autoscaling.targetCPUUtilizationPercentage | int | `80` |  |
| autoscaling.targetMemoryUtilizationPercentage | int | `80` |  |
| cdeventsTarget | string | `""` |  |
| cliIngress.annotations | object | `{}` |  |
| cliIngress.enabled | bool | `false` |  |
| cliIngress.hosts | list | `[]` |  |
| cliIngress.labels | object | `{}` |  |
| cliIngress.path | string | `"/results/(v\\d/.*)"` |  |
| cliIngress.tls | list | `[]` |  |
| cliIngress.tlsenabled | bool | `false` |  |
| cloud.envId | string | `""` |  |
| cloud.existingSecret.envId | string | `""` |  |
| cloud.existingSecret.key | string | `""` |  |
| cloud.existingSecret.name | string | `""` |  |
| cloud.existingSecret.orgId | string | `""` |  |
| cloud.key | string | `""` |  |
| cloud.migrate | string | `""` |  |
| cloud.orgId | string | `""` |  |
| cloud.tls.certificate.caFile | string | `"/tmp/agent-cert/ca.crt"` |  |
| cloud.tls.certificate.certFile | string | `"/tmp/agent-cert/cert.crt"` |  |
| cloud.tls.certificate.keyFile | string | `"/tmp/agent-cert/cert.key"` |  |
| cloud.tls.certificate.secretRef | string | `""` |  |
| cloud.tls.customCaDirPath | string | `""` | Specifies the path to the directory (skip the trailing slash) where CA certificates should be mounted. The mounted file should container a PEM encoded CA certificate. |
| cloud.tls.customCaSecretRef | string | `""` |  |
| cloud.tls.enabled | bool | `true` |  |
| cloud.tls.skipVerify | bool | `false` |  |
| cloud.uiUrl | string | `""` |  |
| cloud.url | string | `"agent.testkube.io:443"` |  |
| clusterName | string | `""` |  |
| configValues | string | `""` |  |
| containerResources | object | `{}` |  |
| dashboardUri | string | `""` |  |
| defaultStorageClassName | string | `""` | Whether to generate RBAC for test job or use manually provided    generateTestJobRBAC: true # default storage class name for PVC volumes |
| disableMongoMigrations | bool | `false` |  |
| disablePostgresMigrations | bool | `false` |  |
| disableSecretCreation | bool | `false` |  |
| dnsPolicy | string | `""` |  |
| dockerImageVersion | string | `""` |  |
| enableK8sEvents | bool | `true` |  |
| enableSecretsEndpoint | bool | `false` |  |
| enabledExecutors | string | `nil` |  |
| executionNamespaces | string | `nil` |  |
| executors | string | `""` |  |
| extraEnvVars | list | `[]` |  |
| fullnameOverride | string | `""` |  |
| global.affinity | object | `{}` |  |
| global.annotations | object | `{}` |  |
| global.features.logsV2 | bool | `false` |  |
| global.features.whitelistedContainers | string | `"init,logs,scraper"` |  |
| global.imagePullSecrets | list | `[]` |  |
| global.imageRegistry | string | `""` |  |
| global.labels | object | `{}` |  |
| global.nodeSelector | object | `{}` |  |
| global.testWorkflows.createOfficialTemplates | bool | `true` |  |
| global.testWorkflows.createServiceAccountTemplates | bool | `true` |  |
| global.testWorkflows.globalTemplate.enabled | bool | `false` |  |
| global.testWorkflows.globalTemplate.external | bool | `false` |  |
| global.testWorkflows.globalTemplate.name | string | `"global-template"` |  |
| global.testWorkflows.globalTemplate.spec | object | `{}` |  |
| global.tls.caCertPath | string | `""` |  |
| global.tolerations | list | `[]` |  |
| global.volumes.additionalVolumeMounts | list | `[]` |  |
| global.volumes.additionalVolumes | list | `[]` |  |
| hostNetwork | string | `""` |  |
| httpReadBufferSize | int | `8192` |  |
| image.digest | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.pullSecrets | list | `[]` |  |
| image.registry | string | `"docker.io"` |  |
| image.repository | string | `"kubeshop/testkube-api-server"` |  |
| imageInspectionCache.enabled | bool | `true` |  |
| imageInspectionCache.name | string | `"testkube-image-cache"` |  |
| imageInspectionCache.ttl | string | `"30m"` |  |
| imageTwInit.digest | string | `""` |  |
| imageTwInit.registry | string | `"docker.io"` |  |
| imageTwInit.repository | string | `"kubeshop/testkube-tw-init"` |  |
| imageTwToolkit.digest | string | `""` |  |
| imageTwToolkit.registry | string | `"docker.io"` |  |
| imageTwToolkit.repository | string | `"kubeshop/testkube-tw-toolkit"` |  |
| initContainerResources | object | `{}` |  |
| jobAnnotations | object | `{}` |  |
| jobContainerTemplate | string | `""` |  |
| jobPodAnnotations | object | `{}` |  |
| jobScraperTemplate | string | `""` |  |
| jobServiceAccountName | string | `""` |  |
| kubeVersion | string | `""` |  |
| livenessProbe.initialDelaySeconds | int | `30` |  |
| logs.bucket | string | `"testkube-logs"` |  |
| logs.storage | string | `"minio"` |  |
| logsServiceAccount.annotations | object | `{}` |  |
| logsServiceAccount.create | bool | `true` |  |
| logsServiceAccount.name | string | `""` |  |
| logsV2ContainerResources | object | `{}` |  |
| minio.accessModes[0] | string | `"ReadWriteOnce"` |  |
| minio.affinity | object | `{}` |  |
| minio.enabled | bool | `true` |  |
| minio.extraEnvVars | list | `[]` |  |
| minio.extraVolumeMounts | list | `[]` |  |
| minio.extraVolumes | list | `[]` |  |
| minio.image.pullPolicy | string | `"IfNotPresent"` |  |
| minio.image.pullSecrets | list | `[]` |  |
| minio.image.registry | string | `"docker.io"` |  |
| minio.image.repository | string | `"minio/minio"` |  |
| minio.image.tag | string | `"RELEASE.2025-07-18T21-56-31Z"` |  |
| minio.livenessProbe.initialDelaySeconds | int | `3` |  |
| minio.livenessProbe.periodSeconds | int | `10` |  |
| minio.matchLabels | list | `[]` |  |
| minio.minioRootPassword | string | `""` |  |
| minio.minioRootUser | string | `""` |  |
| minio.nodeSelector | object | `{}` |  |
| minio.podSecurityContext | object | `{}` |  |
| minio.priorityClassName | string | `""` |  |
| minio.readinessProbe.initialDelaySeconds | int | `3` |  |
| minio.readinessProbe.periodSeconds | int | `10` |  |
| minio.replicaCount | int | `1` |  |
| minio.resources | object | `{}` |  |
| minio.secretPasswordKey | string | `""` |  |
| minio.secretPasswordName | string | `""` |  |
| minio.secretUserKey | string | `""` |  |
| minio.secretUserName | string | `""` |  |
| minio.securityContext | object | `{}` |  |
| minio.serviceAccountName | string | `""` |  |
| minio.serviceMonitor.enabled | bool | `false` |  |
| minio.serviceMonitor.interval | string | `"15s"` |  |
| minio.serviceMonitor.labels | object | `{}` |  |
| minio.serviceMonitor.matchLabels | list | `[]` |  |
| minio.storage | string | `"10Gi"` |  |
| minio.tolerations | list | `[]` |  |
| mongodb.allowDiskUse | bool | `true` |  |
| mongodb.dsn | string | `"mongodb://testkube-mongodb:27017"` |  |
| mongodb.enabled | bool | `true` |  |
| multinamespace.enabled | bool | `false` |  |
| nameOverride | string | `""` |  |
| nats.embedded | bool | `false` |  |
| nats.enabled | bool | `true` |  |
| nats.tls.certSecret.baseMountPath | string | `"/etc/client-certs/nats"` |  |
| nats.tls.certSecret.caFile | string | `"ca.crt"` |  |
| nats.tls.certSecret.certFile | string | `"cert.crt"` |  |
| nats.tls.certSecret.enabled | bool | `false` |  |
| nats.tls.certSecret.keyFile | string | `"cert.key"` |  |
| nats.tls.certSecret.name | string | `"nats-client-cert"` |  |
| nats.tls.enabled | bool | `false` |  |
| nats.tls.mountCACertificate | bool | `false` |  |
| nats.tls.skipVerify | bool | `false` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| podStartTimeout | string | `"30m"` | Testkube timeout for pod start |
| postgresql.dsn | string | `"postgres://testkube:postgres5432@testkube-postgresql:5432/backend?sslmode=disable"` |  |
| postgresql.enabled | bool | `false` |  |
| priorityClassName | string | `""` |  |
| prometheus.enabled | bool | `false` |  |
| prometheus.interval | string | `"15s"` |  |
| prometheus.monitoringLabels | object | `{}` |  |
| rbac.create | bool | `true` |  |
| readinessProbe.initialDelaySeconds | int | `45` |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| scraperContainerResources | object | `{}` |  |
| securityContext | object | `{}` |  |
| service.annotations | object | `{}` |  |
| service.labels | object | `{}` |  |
| service.port | int | `8088` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| slackConfig | string | `""` |  |
| slackSecret | string | `""` |  |
| slackToken | string | `""` |  |
| storage.SSL | bool | `false` |  |
| storage.accessKey | string | `""` |  |
| storage.accessKeyId | string | `""` |  |
| storage.bucket | string | `"testkube-artifacts"` |  |
| storage.certSecret.baseMountPath | string | `"/etc/client-certs/storage"` |  |
| storage.certSecret.caFile | string | `"ca.crt"` |  |
| storage.certSecret.certFile | string | `"cert.crt"` |  |
| storage.certSecret.enabled | bool | `false` |  |
| storage.certSecret.keyFile | string | `"cert.key"` |  |
| storage.certSecret.name | string | `"nats-client-cert"` |  |
| storage.compressArtifacts | bool | `true` |  |
| storage.endpoint | string | `""` |  |
| storage.endpoint_port | string | `"9000"` |  |
| storage.expiration | int | `0` |  |
| storage.mountCACertificate | bool | `false` |  |
| storage.region | string | `""` |  |
| storage.scrapperEnabled | bool | `true` |  |
| storage.secretKeyAccessKeyId | string | `""` |  |
| storage.secretKeySecretAccessKey | string | `""` |  |
| storage.secretNameAccessKeyId | string | `""` |  |
| storage.secretNameSecretAccessKey | string | `""` |  |
| storage.skipVerify | bool | `false` |  |
| storage.token | string | `""` |  |
| storageRequest | string | `"1Gi"` |  |
| templates.job | string | `""` |  |
| templates.jobContainer | string | `""` |  |
| templates.pvcContainer | string | `""` |  |
| templates.scraperContainer | string | `""` |  |
| templates.slavePod | string | `""` |  |
| testConnection.affinity | object | `{}` |  |
| testConnection.enabled | bool | `false` |  |
| testConnection.nodeSelector | object | `{}` |  |
| testConnection.tolerations | list | `[]` |  |
| testServiceAccount.annotations | object | `{}` |  |
| testServiceAccount.create | bool | `true` |  |
| testkubeLogs.grpcAddress | string | `"testkube-logs:9090"` | GRPC address |
| testkubeLogs.tls.certSecret.baseMountPath | string | `"/etc/client-certs/grpc"` | Base path to mount the client certificate secret |
| testkubeLogs.tls.certSecret.caFile | string | `"ca.crt"` | Path to ca file (used for self-signed certificates) |
| testkubeLogs.tls.certSecret.certFile | string | `"cert.crt"` | Path to client certificate file |
| testkubeLogs.tls.certSecret.enabled | bool | `false` | Toggle whether to mount k8s secret which contains GRPC client certificate (cert.crt, cert.key, ca.crt) |
| testkubeLogs.tls.certSecret.keyFile | string | `"cert.key"` | Path to client certificate key file |
| testkubeLogs.tls.certSecret.name | string | `"grpc-client-cert"` | Name of the grpc client certificate secret |
| testkubeLogs.tls.enabled | bool | `false` | Toggle whether to enable TLS in GRPC client |
| testkubeLogs.tls.mountCACertificate | bool | `false` | If enabled, will also require a CA certificate to be provided |
| testkubeLogs.tls.skipVerify | bool | `false` | Toggle whether to verify certificates |
| tolerations | list | `[]` |  |
| uiIngress.annotations | object | `{}` |  |
| uiIngress.enabled | bool | `false` |  |
| uiIngress.hosts | list | `[]` |  |
| uiIngress.labels | object | `{}` |  |
| uiIngress.path | string | `"/results/(v\\d/executions.*)"` |  |
| uiIngress.pathType | string | `"Prefix"` |  |
| uiIngress.tls | list | `[]` |  |
| uiIngress.tlsenabled | bool | `false` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.13.1](https://github.com/norwoodj/helm-docs/releases/v1.13.1)

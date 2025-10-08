# testkube

Testkube is an open-source platform that simplifies the deployment and management of automated testing infrastructure.

![Version: 2.0.17](https://img.shields.io/badge/Version-2.0.17-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

## Install

Add `kubeshop` Helm repository and fetch latest charts info:

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts
helm repo update
```

### TLS

Testkube API needs to have additional configuration if NATS or MinIO (or any
S3-compatible storage) is used.  THe following sections describe how to
configure TLS for NATS and MinIO.

#### NATS

If you want to provision NATS server with TLS, first you will need to create a
Kubernetes secret which contains the server certificate, certificate key and CA
certificate, and then you can use the following configuration

```yaml
nats:
  nats:
    tls:
      allowNonTLS: false
      secret:
        name: nats-server-cert
      ca: "ca.crt"
      cert: "cert.crt"
      key: "cert.key"
```

If NATS is configured to use TLS, Testkube API needs to set the `secure` flag so
it uses a secure protocol when connecting to NATS.

```yaml
testkube-api:
  nats:
    tls:
      enabled: true
```

Additionally, if NATS is configured to use a self-signed certificate, Testkube API needs to have the CA & client certificate in order to verify the NATS server certificate.
You will need to create a Kubernetes secret which contains the client certificate, certificate key and CA certificate.

```yaml
testkube-api:
  nats:
    tls:
      enabled: true
      mountCACertificate: true
      certSecret:
        enabled: true
```

It is also possible to skip the verification of the NATS server certificate by setting the `skipVerify` flag to `true`.

```yaml
testkube-api:
  nats:
    tls:
      enabled: true
      skipVerify: true
```
To use external NATS server, it's possible to configure:

```yaml
testkube-api:
  nats:
    enabled: false
    uri: nats://some-nats-address:4222
    # or providing URI with Kubernetes secret:
    # secretName: example-secret
    # secretKey: example-key

```
#### MinIO/S3

Currently, Testkube doesn't support provisioning MinIO with TLS. However, if you use an external MinIO (or any S3-compatible storage)
you can configure Testkube API to use TLS when connecting to it.

```yaml
testkube-api:
  storage:
    SSL: true
```

Additionally, if S3 server is configured to use a self-signed certificate, Testkube API needs to have the CA & client certificate in order to verify the S3 server certificate.
You will need to create a Kubernetes secret which contains the client certificate, certificate key and CA certificate.

```yaml
testkube-api:
  storage:
    SSL: true
    mountCACertificate: true
    certSecret:
      enabled: true
```

It is also possible to skip the verification of the S3 server certificate by setting the `skipVerify` flag to `true`.

```yaml
testkube-api:
  storage:
    SSL: true
    skipVerify: true
```

NOTE:
This will add CustomResourceDefinitions and RBAC roles and RoleBindings to the cluster.
This installation requires having cluster administrative rights.

```sh
helm install testkube kubeshop/testkube --create-namespace --namespace testkube
```

## Uninstall

NOTE: Uninstalling Testkube will also delete all CRDs and all resources created by Testkube.

```sh
helm delete testkube -n testkube
kubectl delete namespace testkube
```

## Migration to upgradable CRDs Helm chart

Originally Helm chart stored CRDs in a special crds/ folder. In order to make them upgradable they were moved
into the regular templates/ folder. Unfortunately Helm uses different annotations and labels for resources located
in crds/ and templates/ folders. Please run these commands to fix it:

```sh
kubectl annotate --overwrite crds executors.executor.testkube.io meta.helm.sh/release-name="testkube" meta.helm.sh/release-namespace="testkube"
kubectl annotate --overwrite crds tests.tests.testkube.io meta.helm.sh/release-name="testkube" meta.helm.sh/release-namespace="testkube"
kubectl annotate --overwrite crds scripts.tests.testkube.io meta.helm.sh/release-name="testkube" meta.helm.sh/release-namespace="testkube"
kubectl label --overwrite crds executors.executor.testkube.io app.kubernetes.io/managed-by=Helm
kubectl label --overwrite crds tests.tests.testkube.io app.kubernetes.io/managed-by=Helm
kubectl label --overwrite crds scripts.tests.testkube.io app.kubernetes.io/managed-by=Helm
```

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| file://../testkube-api | testkube-api | 2.0.10 |
| file://../testkube-logs | testkube-logs | 0.2.0 |
| file://../testkube-operator | testkube-operator | 2.0.0 |
| https://charts.bitnami.com/bitnami | mongodb | 13.10.1 |
| https://nats-io.github.io/k8s/helm/charts/ | nats | 1.1.7 |
| https://charts.bitnami.com/bitnami | postgresql | 16.3.0 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| global | object | `{"affinity":{},"annotations":{},"features":{"logsV2":false,"whitelistedContainers":"init,logs,scraper"},"imagePullSecrets":[],"imageRegistry":"","labels":{},"nodeSelector":{},"testWorkflows":{"createOfficialTemplates":true,"createServiceAccountTemplates":true,"globalTemplate":{"enabled":false,"name":"global-template","spec":{}}},"tls":{"caCertPath":""},"tolerations":[{"effect":"NoSchedule","key":"kubernetes.io/arch","operator":"Equal","value":"arm64"}],"volumes":{"additionalVolumeMounts":[],"additionalVolumes":[]}}` | Important! Please, note that this will override sub-chart image parameters. |
| global.affinity | object | `{}` | Affinity rules to add to all deployed Pods |
| global.annotations | object | `{}` | Annotations to add to all deployed objects |
| global.features.logsV2 | bool | `false` | Toggle whether to enable V2 log support |
| global.features.whitelistedContainers | string | `"init,logs,scraper"` | Comma-separated array of containers which should get scraped for logs |
| global.imagePullSecrets | list | `[]` | Global Docker registry secret names as an array |
| global.imageRegistry | string | `""` | Global Docker image registry |
| global.labels | object | `{}` | Labels to add to all deployed objects |
| global.nodeSelector | object | `{}` | Node labels to add to all deployed Pods |
| global.testWorkflows | object | `{"createOfficialTemplates":true,"createServiceAccountTemplates":true,"globalTemplate":{"enabled":false,"name":"global-template","spec":{}}}` | Test Workflows configuration |
| global.testWorkflows.createOfficialTemplates | bool | `true` | Create TestWorkflowTemplates with automatically configured execution |
| global.testWorkflows.createServiceAccountTemplates | bool | `true` | Create TestWorkflowTemplates to easily use the service account |
| global.testWorkflows.globalTemplate | object | `{"enabled":false,"name":"global-template","spec":{}}` | Global TestWorkflowTemplate that will be automatically included for all executions |
| global.testWorkflows.globalTemplate.enabled | bool | `false` | Is global template enabled |
| global.testWorkflows.globalTemplate.name | string | `"global-template"` | Name of the global template |
| global.testWorkflows.globalTemplate.external | bool | `false` | Is the global template sourced externally? (otherwise it's created from spec below) |
| global.testWorkflows.globalTemplate.spec | object | `{}` | Specification for the global template |
| global.tls.caCertPath | string | `""` | Path to the PEM-encoded CA certificate file (needs to be mounted to the container previously) |
| global.tolerations | list | `[{"effect":"NoSchedule","key":"kubernetes.io/arch","operator":"Equal","value":"arm64"}]` | Tolerations to add to all deployed pods |
| global.volumes | object | `{"additionalVolumeMounts":[],"additionalVolumes":[]}` | Global volume settings (API & Test Jobs) |
| global.volumes.additionalVolumeMounts | list | `[]` | Additional volume mounts to be added to the Testkube API container and Test Jobs containers |
| global.volumes.additionalVolumes | list | `[]` | Additional volumes to be added to the Testkube API container and Test Jobs containers |
| mongodb.auth.enabled | bool | `false` | Toggle whether to enable MongoDB authentication |
| mongodb.containerSecurityContext | object | `{}` | Security Context for MongoDB container |
| mongodb.enabled | bool | `true` | Toggle whether to install MongoDB |
| mongodb.fullnameOverride | string | `"testkube-mongodb"` | MongoDB fullname override |
| mongodb.image.pullSecrets | list | `[]` | MongoDB image pull Secret |
| mongodb.image.registry | string | `"docker.io"` | MongoDB image registry |
| mongodb.image.repository | string | `"zcube/bitnami-compat-mongodb"` | MongoDB image repository |
| mongodb.image.tag | string | `"6.0.5-debian-11-r64"` | MongoDB image tag |
| mongodb.livenessProbe.enabled | bool | `true` |  |
| mongodb.livenessProbe.failureThreshold | int | `6` |  |
| mongodb.livenessProbe.initialDelaySeconds | int | `30` |  |
| mongodb.livenessProbe.periodSeconds | int | `240` |  |
| mongodb.livenessProbe.successThreshold | int | `1` |  |
| mongodb.livenessProbe.timeoutSeconds | int | `10` |  |
| mongodb.podSecurityContext | object | `{}` | MongoDB Pod Security Context |
| mongodb.readinessProbe | object | `{"enabled":true,"failureThreshold":6,"initialDelaySeconds":5,"periodSeconds":240,"successThreshold":1,"timeoutSeconds":5}` | Settings for readiness and liveness probes |
| mongodb.resources | object | `{"requests":{"cpu":"150m","memory":"100Mi"}}` | MongoDB resource settings |
| mongodb.service | object | `{"clusterIP":"","nodePort":true,"port":"27017","portName":"mongodb"}` | MongoDB service settings |
| nats.config.jetstream.enabled | bool | `true` | Toggle whether to enable JetStream (should not be disabled as Testkube uses Jetstream features) |
| nats.config.merge.max_payload | string | `"<< 8MB >>"` |  |
| nats.natsBox | object | `{"enabled":false}` | Uncomment to override the NATS Server image options container:   image:     repository: nats     tag: 2.11.6-alpine     pullPolicy:     registry: NATS Box container settings TODO remove this container after tests on dev and stage nats-box is A lightweight container with NATS utilities. It's not needed for nats server change it to natsBox:   enabled: false |
| nats.reloader.enabled | bool | `true` |  |
| postgresql.architecture | string | `"standalone"` | PostgreSQL architecture |
| postgresql.auth.enabledPostgresUser | bool | `true` | Enable "postgres" admin user for PostgreSQL |
| postgresql.auth.database | string | `backend` | Name for a custom database to create in PostgreSQL |
| postgresql.auth.password | string | `postgres5432` | Password for the custom user to create in PostgreSQL |
| postgresql.auth.postgresPassword | string | `postgres1234` | Password for the "postgres" admin user in PostgreSQL |
| postgresql.fullnameOverride | string | `"testkube-postgresql"` | PostgeSQL fullname override |
| postgresql.enabled | bool | `false` | Toggle whether to install PostgreSQL |
| postgresql.global.security.allowInsecureImages | bool | `true` | Allows skipping image verification for PostgreSQL |
| postgresql.image.pullSecrets | list | `[]` | PostgreSQL image pull Secret |
| postgresql.image.registry | string | `"docker.io"` | PostgreSQL image registry |
| postgresql.image.repository | string | `"zcube/bitnami-compat-postgresql"` | PostgreSQL image repository |
| postgresql.image.tag | string | `"15.2.0-debian-11-r64"` | PostgreSQL image tag |
| postgresql.primary.configuration | string | `""` | Configuration for PostgreSQL |
| postgresql.primary.containerSecurityContext | object | `{}` | Security Context for PostgreSQL container |
| postgresql.primary.extraVolumes| list | `[{"name": "postgresql-run", "emptyDir": {}}]` | PostgreSQL extra volumes for writable directories
| postgresql.primary.extraVolumeMounts| list | `[{"name": "postgresql-run", "mountPath": "/var/run/postgresql"}]` | PostgreSQL extra volume mounts for writable directories
| postgresql.primary.livenessProbe.enabled | bool | `true` | Settings for liveness probes |
| postgresql.primary.livenessProbe.failureThreshold | int | `6` |  |
| postgresql.primary.livenessProbe.initialDelaySeconds | int | `30` |  |
| postgresql.primary.livenessProbe.periodSeconds | int | `10` |  |
| postgresql.primary.livenessProbe.timeoutSeconds | int | `5` |  |
| postgresql.primary.readinessProbe.enabled | bool | `true` | Settings for readiness probes |
| postgresql.primary.readinessProbe.failureThreshold | int | `6` |  |
| postgresql.primary.readinessProbe.initialDelaySeconds | int | `5` |  |
| postgresql.primary.readinessProbe.periodSeconds | int | `10` |  |
| postgresql.primary.readinessProbe.timeoutSeconds | int | `5` |  |
| postgresql.primary.readinessProbe.successThreshold | int | `6` |  |
| postgresql.primary.resources | object | `{"requests":{"cpu":"150m","memory":"100Mi"}}` | PostgreSQL resource settings |
| postgresql.primary.persistence.enabled | bool | `true` | Enable persistence for PostgreSQL |
| postgresql.primary.persistence.size | string | `1Gi` | Size for PostgreSQL volume |
| postgresql.primary.persistence.storageClass | string | `""` | Storage Class for PostgreSQL |
| postgresql.primary.podSecurityContext | object | `{}` | PostgreSQL Pod Security Context |
| postgresql.primary.service | object | `{"clusterIP":"","ports":{"postgresql": 5432},"type":"NodePort"}` | PostgreSQL service settings |
| preUpgradeHook | object | `{"annotations":{},"enabled":true,"image":{"pullPolicy":"IfNotPresent","pullSecrets":[],"registry":"docker.io","repository":"bitnami/kubectl","tag":"1.28.2"},"labels":{},"name":"mongodb-upgrade","nodeSelector":{},"podAnnotations":{},"podSecurityContext":{},"resources":{},"securityContext":{},"serviceAccount":{"create":true},"tolerations":[],"ttlSecondsAfterFinished":100}` | MongoDB pre-upgrade parameters |
| preUpgradeHook.enabled | bool | `true` | Upgrade hook is enabled |
| preUpgradeHook.image | object | `{"pullPolicy":"IfNotPresent","pullSecrets":[],"registry":"docker.io","repository":"bitnami/kubectl","tag":"1.28.2"}` | Specify image |
| preUpgradeHook.name | string | `"mongodb-upgrade"` | Upgrade hook name |
| preUpgradeHook.nodeSelector | object | `{}` | Node labels for pod assignment. |
| preUpgradeHook.podSecurityContext | object | `{}` | MongoDB Upgrade Pod Security Context |
| preUpgradeHook.resources | object | `{}` | Specify resource limits and requests |
| preUpgradeHook.securityContext | object | `{}` | Security Context for MongoDB Upgrade kubectl container |
| preUpgradeHook.serviceAccount | object | `{"create":true}` | Create SA for upgrade hook |
| preUpgradeHook.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| preUpgradeHookNATS | object | `{"annotations":{},"enabled":true,"image":{"pullPolicy":"IfNotPresent","pullSecrets":[],"registry":"docker.io","repository":"bitnami/kubectl","tag":"1.28.2"},"labels":{},"name":"nats-upgrade","nodeSelector":{},"podAnnotations":{},"podSecurityContext":{},"resources":{},"securityContext":{},"serviceAccount":{"create":true},"tolerations":[],"ttlSecondsAfterFinished":100}` | NATS pre-upgrade parameters |
| preUpgradeHookNATS.enabled | bool | `true` | Upgrade hook is enabled |
| preUpgradeHookNATS.image | object | `{"pullPolicy":"IfNotPresent","pullSecrets":[],"registry":"docker.io","repository":"bitnami/kubectl","tag":"1.28.2"}` | Specify image |
| preUpgradeHookNATS.name | string | `"nats-upgrade"` | Upgrade hook name |
| preUpgradeHookNATS.nodeSelector | object | `{}` | Node labels for pod assignment. |
| preUpgradeHookNATS.podSecurityContext | object | `{}` | NATS Upgrade Pod Security Context |
| preUpgradeHookNATS.resources | object | `{}` | Specify resource limits and requests |
| preUpgradeHookNATS.securityContext | object | `{}` | Security Context for NATS Upgrade kubectl container |
| preUpgradeHookNATS.serviceAccount | object | `{"create":true}` | Create SA for upgrade hook |
| preUpgradeHookNATS.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-api.additionalJobVolumeMounts | list | `[]` | Additional volume mounts to be added to the Test Jobs |
| testkube-api.additionalJobVolumes | list | `[]` | Additional volumes to be added to the Test Jobs |
| testkube-api.additionalNamespaces | list | `[]` | Watch namespaces. In this case, a Role and a RoleBinding will be created for each specified namespace. |
| testkube-api.additionalVolumeMounts | list | `[]` | Additional volume mounts to be added |
| testkube-api.additionalVolumes | list | `[]` | Additional volumes to be added |
| testkube-api.allowLowSecurityFields |  bool | `false` | Allow to use low securiy fields for test workflow pod and container configurations
| testkube-api.analyticsEnabled | bool | `true` | Enable analytics for Testkube |
| testkube-api.cdeventsTarget | string | `""` | target for cdevents emission via http(s) |
| testkube-api.cliIngress.annotations | object | `{}` | Additional annotations for the Ingress resource. |
| testkube-api.cliIngress.enabled | bool | `false` | Use ingress |
| testkube-api.cliIngress.hosts | list | `["testkube.example.com"]` | Hostnames must be provided if Ingress is enabled. |
| testkube-api.cliIngress.path | string | `"/results/(v\\d/.*)"` |  |
| testkube-api.cliIngress.tls | list | `[]` | Placing a host in the TLS config will indicate a certificate should be created |
| testkube-api.cliIngress.tlsenabled | bool | `false` | Toggle whether to enable TLS on the ingress |
| testkube-api.cloud.key | string | `""` | Testkube Clouc License Key (for Environment) |
| testkube-api.cloud.tls.certificate.caFile | string | `"/tmp/agent-cert/ca.crt"` | Default path for ca file |
| testkube-api.cloud.tls.certificate.certFile | string | `"/tmp/agent-cert/cert.crt"` | Default path for certificate file |
| testkube-api.cloud.tls.certificate.keyFile | string | `"/tmp/agent-cert/cert.key"` | Default path for certificate key file |
| testkube-api.cloud.tls.certificate.secretRef | string | `""` | When provided, it will use the provided certificates when authenticating with the Agent (gRPC) API (secret should contain cert.crt, key.crt and ca.crt) |
| testkube-api.cloud.tls.customCaDirPath | string | `""` | Specifies the path to the directory (skip the trailing slash) where CA certificates should be mounted. The mounted file should container a PEM encoded CA certificate. |
| testkube-api.cloud.tls.customCaSecretRef | string | `""` |  |
| testkube-api.cloud.tls.enabled | bool | `true` | Toggle should the connection to Agent API in Cloud/Enterprise use secure GRPC (GRPCS) (if false, it will use insecure GRPC) |
| testkube-api.cloud.tls.skipVerify | bool | `false` | Toggle should the client skip verifying the Agent API server cert in Cloud/Enterprise |
| testkube-api.cloud.uiUrl | string | `""` |  |
| testkube-api.cloud.url | string | `"agent.testkube.io:443"` | Testkube Cloud API URL |
| testkube-api.clusterName | string | `""` | cluster name to be used in events |
| testkube-api.containerResources | object | `{}` |  |
| testkube-api.dashboardUri | string | `""` | dashboard uri to be used in notification events |
| testkube-api.defaultStorageClassName | string | `""` | default storage class name for PVC volumes |
| testkube-api.disableSecretCreation | bool | `false` | disable secret creation for tests and test sources |
| testkube-api.dnsPolicy | string | `""` | Specify dnsPolicy for Testkube API Deployment |
| testkube-api.dockerImageVersion | string | "" | dockerImageVersion of Testkube Agent |
| testkube-api.enableK8sEvents | bool | `true` | enable k8s events for testkube events |
| testkube-api.enableSecretsEndpoint | bool | `false` | enable endpoint to list testkube namespace secrets |
| testkube-api.enabledExecutors | string | `nil` | enable only specified executors with enabled flag |
| testkube-api.executionNamespaces | list | `[]` | Execution namespaces for Testkube API to only run tests In this case, a Role and a RoleBinding will be created for each specified namespace. |
| testkube-api.executors | string | `""` | default executors as base64-encoded string |
| testkube-api.extraEnvVars | list | `[]` | Extra environment variables to be set on deployment |
| testkube-api.fullnameOverride | string | `"testkube-api-server"` | Testkube API full name override |
| testkube-api.hostNetwork | string | `""` | Specify hostNetwork for Testkube API Deployment |
| testkube-api.image.digest | string | `""` | Testkube API image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag |
| testkube-api.image.pullPolicy | string | `"IfNotPresent"` | Testkube API image tag |
| testkube-api.image.pullSecrets | list | `[]` | Testkube API k8s secret for private registries |
| testkube-api.image.registry | string | `"docker.io"` | Testkube API image registry |
| testkube-api.image.repository | string | `"kubeshop/testkube-api-server"` | Testkube API image name |
| testkube-api.imageInspectionCache.enabled | bool | `true` | Status of the persistent cache |
| testkube-api.imageInspectionCache.name | string | `"testkube-image-cache"` | ConfigMap name to persist cache |
| testkube-api.imageInspectionCache.ttl | string | `"30m"` | TTL for image pull secrets cache (set to 0 to disable) |
| testkube-api.imageTwInit.digest | string | `""` | Test Workflows image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag |
| testkube-api.imageTwInit.pullSecrets | list | `[]` | Test Workflows image k8s secret for private registries |
| testkube-api.imageTwInit.registry | string | `"docker.io"` | Test Workflows image registry |
| testkube-api.imageTwInit.repository | string | `"kubeshop/testkube-tw-init"` | Test Workflows image name |
| testkube-api.imageTwToolkit.digest | string | `""` | Test Workflows image digest in the way sha256:aa.... Please note this parameter, if set, will override the tag |
| testkube-api.imageTwToolkit.registry | string | `"docker.io"` | Test Workflows image registry |
| testkube-api.imageTwToolkit.repository | string | `"kubeshop/testkube-tw-toolkit"` | Test Workflows image name |
| testkube-api.initContainerResources | object | `{}` |  |
| testkube-api.jobAnnotations | object | `{}` |  |
| testkube-api.jobPodAnnotations | object | `{}` |  |
| testkube-api.jobServiceAccountName | string | `""` | SA that is used by a job. Can be annotated with the IAM Role Arn to access S3 service in AWS Cloud. |
| testkube-api.livenessProbe | object | `{"initialDelaySeconds":15}` | Testkube API Liveness probe parameters |
| testkube-api.livenessProbe.initialDelaySeconds | int | `15` | Initial delay for liveness probe |
| testkube-api.logs.bucket | string | `"testkube-logs"` | Bucket should be specified if storage is "minio" |
| testkube-api.logs.storage | string | `"minio"` | Log storage can either be "minio" or "mongo" |
| testkube-api.logsV2ContainerResources | object | `{}` |  |
| testkube-api.minio.accessModes | list | `["ReadWriteOnce"]` | PVC Access Modes for Minio. The volume is mounted as read-write by a single node. |
| testkube-api.minio.affinity | object | `{}` | Affinity for pod assignment. |
| testkube-api.minio.enabled | bool | `true` | Toggle whether to install MinIO |
| testkube-api.minio.extraEnvVars | list | `[]` | Minio extra vars |
| testkube-api.minio.extraVolumeMounts | list | `[]` |  |
| testkube-api.minio.extraVolumes | list | `[]` |  |
| testkube-api.minio.image | object | `{"pullSecrets":[],"registry":"docker.io","repository":"minio/minio","tag":"RELEASE.2025-07-18T21-56-31Z"}` | Minio image from DockerHub |
| testkube-api.minio.minioRootPassword | string | `"minio123"` | Root password |
| testkube-api.minio.minioRootUser | string | `"minio"` | Root username |
| testkube-api.minio.nodeSelector | object | `{}` | Node labels for pod assignment. |
| testkube-api.minio.podSecurityContext | object | `{}` | MinIO Pod Security Context |
| testkube-api.minio.priorityClassName | string | `""` |  |
| testkube-api.minio.resources | object | `{}` | MinIO Resources settings |
| testkube-api.minio.secretPasswordKey | string | `""` |  |
| testkube-api.minio.secretPasswordName | string | `""` |  |
| testkube-api.minio.secretUserKey | string | `""` |  |
| testkube-api.minio.secretUserName | string | `""` |  |
| testkube-api.minio.securityContext | object | `{}` | Security Context for MinIO container |
| testkube-api.minio.serviceAccountName | string | `""` | ServiceAccount name to use for Minio |
| testkube-api.minio.serviceMonitor.enabled | bool | `false` |  |
| testkube-api.minio.serviceMonitor.interval | string | `"15s"` |  |
| testkube-api.minio.serviceMonitor.labels | object | `{}` |  |
| testkube-api.minio.serviceMonitor.matchLabels | list | `[]` |  |
| testkube-api.minio.storage | string | `"10Gi"` | PVC Storage Request for MinIO. Should be available in the cluster. |
| testkube-api.minio.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-api.mongodb.allowDiskUse | bool | `true` | Allow or prohibit writing temporary files on disk when a pipeline stage exceeds the 100 megabyte limit. |
| testkube-api.mongodb.dsn | string | `"mongodb://testkube-mongodb:27017"` | MongoDB DSN |
| testkube-api.mongodb.enabled | bool | `true` | use MongoDB |
| testkube-api.multinamespace.enabled | bool | `false` |  |
| testkube-api.nameOverride | string | `"api-server"` | Testkube API name override |
| testkube-api.nats.embedded | bool | `false` | Start NATS embedded server in api binary instead of separate deployment |
| testkube-api.nats.enabled | bool | `true` | Use NATS |
| testkube-api.nats.tls.certSecret.baseMountPath | string | `"/etc/client-certs/storage"` | Base path to mount the client certificate secret |
| testkube-api.nats.tls.certSecret.caFile | string | `"ca.crt"` | Path to ca file (used for self-signed certificates) |
| testkube-api.nats.tls.certSecret.certFile | string | `"cert.crt"` | Path to client certificate file |
| testkube-api.nats.tls.certSecret.enabled | bool | `false` | Toggle whether to mount k8s secret which contains storage client certificate (cert.crt, cert.key, ca.crt) |
| testkube-api.nats.tls.certSecret.keyFile | string | `"cert.key"` | Path to client certificate key file |
| testkube-api.nats.tls.certSecret.name | string | `"nats-client-cert"` | Name of the storage client certificate secret |
| testkube-api.nats.tls.enabled | bool | `false` | Toggle whether to enable TLS in NATS client |
| testkube-api.nats.tls.mountCACertificate | bool | `false` | If enabled, will also require a CA certificate to be provided |
| testkube-api.nats.tls.skipVerify | bool | `false` | Toggle whether to verify certificates |
| testkube-api.nats.uri | string | `"nats://testkube-nats:4222"` | NATS URI |
| testkube-api.podSecurityContext | object | `{}` | Testkube API Pod Security Context |
| testkube-api.podStartTimeout | string | `"30m"` | Testkube timeout for pod start |
| testkube-api.priorityClassName | string | `""` |  |
| testkube-api.prometheus.enabled | bool | `false` | Use monitoring |
| testkube-api.prometheus.interval | string | `"15s"` | Scrape interval |
| testkube-api.prometheus.monitoringLabels | object | `{}` | The name of the label to use in serviceMonitor if Prometheus is enabled |
| testkube-api.rbac | object | `{"create":true}` | Toggle whether to deploy Testkube API RBAC |
| testkube-api.readinessProbe | object | `{"initialDelaySeconds":30}` | Testkube API Readiness probe parameters |
| testkube-api.readinessProbe.initialDelaySeconds | int | `30` | Initial delay for readiness probe |
| testkube-api.resources | object | `{"requests":{"cpu":"200m","memory":"200Mi"}}` | Testkube API resource requests and limits |
| testkube-api.scraperContainerResources | object | `{}` |  |
| testkube-api.securityContext | object | `{}` | Security Context for testkube-api container |
| testkube-api.service.annotations | object | `{}` | Service Annotations |
| testkube-api.service.labels | object | `{}` | Service labels |
| testkube-api.service.port | int | `8088` | HTTP Port |
| testkube-api.service.type | string | `"ClusterIP"` | Adapter service type for working with real k8s we should use "ClusterIP" type. |
| testkube-api.serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| testkube-api.serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| testkube-api.serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template. |
| testkube-api.slackConfig | string | `nil` | Slack config for the events, tests, testsuites, testworkflows and channels |
| testkube-api.slackSecret | string | `""` | Slack secret to store slackToken, the key name should be SLACK_TOKEN |
| testkube-api.slackToken | string | `""` | Slack token from the testkube authentication endpoint |
| testkube-api.storage.SSL | bool | `false` | MinIO Use SSL |
| testkube-api.storage.accessKey | string | `"minio123"` | MinIO Secret Access Key |
| testkube-api.storage.accessKeyId | string | `"minio"` | MinIO Access Key ID |
| testkube-api.storage.bucket | string | `"testkube-artifacts"` | MinIO Bucket |
| testkube-api.storage.certSecret.baseMountPath | string | `"/etc/client-certs/storage"` | Base path to mount the client certificate secret |
| testkube-api.storage.certSecret.caFile | string | `"ca.crt"` | Path to ca file (used for self-signed certificates) |
| testkube-api.storage.certSecret.certFile | string | `"cert.crt"` | Path to client certificate file |
| testkube-api.storage.certSecret.enabled | bool | `false` | Toggle whether to mount k8s secret which contains storage client certificate (cert.crt, cert.key, ca.crt) |
| testkube-api.storage.certSecret.keyFile | string | `"cert.key"` | Path to client certificate key file |
| testkube-api.storage.certSecret.name | string | `"storage-client-cert"` | Name of the storage client certificate secret |
| testkube-api.storage.compressArtifacts | bool | `true` | Toggle whether to compress artifacts in Testkube API |
| testkube-api.storage.endpoint | string | `""` | MinIO endpoint |
| testkube-api.storage.endpoint_port | string | `"9000"` | MinIO endpoint port |
| testkube-api.storage.expiration | int | `0` | MinIO Expiration period in days |
| testkube-api.storage.mountCACertificate | bool | `false` | If enabled, will also require a CA certificate to be provided |
| testkube-api.storage.region | string | `""` | MinIO Region |
| testkube-api.storage.scrapperEnabled | bool | `true` | Toggle whether to enable scraper in Testkube API |
| testkube-api.storage.secretKeyAccessKeyId | string | `""` | Key for storage accessKeyId taken from k8s secret |
| testkube-api.storage.secretKeySecretAccessKey | string | `""` | Key for storage secretAccessKeyId taken from k8s secret |
| testkube-api.storage.secretNameAccessKeyId | string | `""` | k8s Secret name for storage accessKeyId |
| testkube-api.storage.secretNameSecretAccessKey | string | `""` | K8s Secret Name for storage secretAccessKeyId |
| testkube-api.storage.skipVerify | bool | `false` | Toggle whether to verify TLS certificates |
| testkube-api.storage.token | string | `""` | MinIO Token |
| testkube-api.storageRequest | string | `"1Gi"` |  |
| testkube-api.testConnection.enabled | bool | `false` | Toggle whether to create Test Connection pod |
| testkube-api.testConnection.resources | object | `{}` | Test Connection resource settings |
| testkube-api.testConnection.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-api.testServiceAccount | object | `{"annotations":{},"create":true}` | Service Account parameters |
| testkube-api.testServiceAccount.annotations | object | `{}` | Annotations to add to the service account |
| testkube-api.testServiceAccount.create | bool | `true` | Specifies whether a service account should be created |
| testkube-api.testkubeLogs.grpcAddress | string | `"testkube-logs:9090"` | GRPC address |
| testkube-api.testkubeLogs.tls.certSecret.baseMountPath | string | `"/etc/client-certs/grpc"` | Base path to mount the client certificate secret |
| testkube-api.testkubeLogs.tls.certSecret.caFile | string | `"ca.crt"` | Path to ca file (used for self-signed certificates) |
| testkube-api.testkubeLogs.tls.certSecret.certFile | string | `"cert.crt"` | Path to client certificate file |
| testkube-api.testkubeLogs.tls.certSecret.enabled | bool | `false` | Toggle whether to mount k8s secret which contains GRPC client certificate (cert.crt, cert.key, ca.crt) |
| testkube-api.testkubeLogs.tls.certSecret.keyFile | string | `"cert.key"` | Path to client certificate key file |
| testkube-api.testkubeLogs.tls.certSecret.name | string | `"grpc-client-cert"` | Name of the grpc client certificate secret |
| testkube-api.testkubeLogs.tls.enabled | bool | `false` | Toggle whether to enable TLS in GRPC client |
| testkube-api.testkubeLogs.tls.mountCACertificate | bool | `false` | If enabled, will also require a CA certificate to be provided |
| testkube-api.testkubeLogs.tls.skipVerify | bool | `false` | Toggle whether to verify certificates |
| testkube-api.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-api.uiIngress.annotations | object | `{}` | Additional annotations for the Ingress resource. |
| testkube-api.uiIngress.enabled | bool | `false` | Use Ingress |
| testkube-api.uiIngress.hosts | list | `["testkube.example.com"]` | Hostnames must be provided if Ingress is enabled. |
| testkube-api.uiIngress.path | string | `"/results/(v\\d/.*)"` |  |
| testkube-api.uiIngress.tls | list | `[]` | Placing a host in the TLS config will indicate a certificate should be created |
| testkube-api.uiIngress.tlsenabled | bool | `false` |  |
| testkube-logs.fullnameOverride | string | `"testkube-logs"` | Testkube logs full name override |
| testkube-logs.nameOverride | string | `"logs"` | Testkube logs name override |
| testkube-logs.replicaCount | int | `1` |  |
| testkube-logs.storage.SSL | bool | `false` | MinIO Use SSL |
| testkube-logs.storage.accessKey | string | `"minio123"` | MinIO Secret Access Key |
| testkube-logs.storage.accessKeyId | string | `"minio"` | MinIO Access Key ID |
| testkube-logs.storage.bucket | string | `"testkube-logs"` | MinIO Bucket |
| testkube-logs.storage.certSecret.baseMountPath | string | `"/etc/client-certs/storage"` | Base path to mount the client certificate secret |
| testkube-logs.storage.certSecret.caFile | string | `"ca.crt"` | Path to ca file (used for self-signed certificates) |
| testkube-logs.storage.certSecret.certFile | string | `"cert.crt"` | Path to client certificate file |
| testkube-logs.storage.certSecret.enabled | bool | `false` | Toggle whether to mount k8s secret which contains storage client certificate (cert.crt, cert.key, ca.crt) |
| testkube-logs.storage.certSecret.keyFile | string | `"cert.key"` | Path to client certificate key file |
| testkube-logs.storage.certSecret.name | string | `"storage-client-cert"` | Name of the storage client certificate secret |
| testkube-logs.storage.compressArtifacts | bool | `true` | Toggle whether to compress artifacts in Testkube API |
| testkube-logs.storage.endpoint | string | `""` | MinIO endpoint |
| testkube-logs.storage.endpoint_port | string | `"9000"` | MinIO endpoint port |
| testkube-logs.storage.expiration | int | `0` | MinIO Expiration period in days |
| testkube-logs.storage.mountCACertificate | bool | `false` | If enabled, will also require a CA certificate to be provided |
| testkube-logs.storage.region | string | `""` | MinIO Region |
| testkube-logs.storage.scrapperEnabled | bool | `true` | Toggle whether to enable scraper in Testkube API |
| testkube-logs.storage.secretKeyAccessKeyId | string | `""` | Key for storage accessKeyId taken from k8s secret |
| testkube-logs.storage.secretKeySecretAccessKey | string | `""` | Key for storage secretAccessKeyId taken from k8s secret |
| testkube-logs.storage.secretNameAccessKeyId | string | `""` | k8s Secret name for storage accessKeyId |
| testkube-logs.storage.secretNameSecretAccessKey | string | `""` | K8s Secret Name for storage secretAccessKeyId |
| testkube-logs.storage.skipVerify | bool | `false` | Toggle whether to verify TLS certificates |
| testkube-logs.storage.token | string | `""` | MinIO Token |
| testkube-logs.testConnection | object | `{"enabled":false}` | Test Connection pod |
| testkube-logs.tls.certSecret.baseMountPath | string | `"/etc/server-certs/grpc"` | Base path to mount the server certificate secret |
| testkube-logs.tls.certSecret.certFile | string | `"cert.crt"` | Path to server certificate file |
| testkube-logs.tls.certSecret.clientCAFile | string | `"client_ca.crt"` | Path to client ca file (used for self-signed certificates) |
| testkube-logs.tls.certSecret.enabled | bool | `false` | Toggle whether to mount k8s secret which contains GRPC server certificate (cert.crt, cert.key, client_ca.crt) |
| testkube-logs.tls.certSecret.keyFile | string | `"cert.key"` | Path to server certificate key file |
| testkube-logs.tls.certSecret.name | string | `"grpc-server-cert"` | Name of the grpc server certificate secret |
| testkube-logs.tls.clientAuth | bool | `false` | Toggle whether to require client auth |
| testkube-logs.tls.enabled | bool | `false` | Toggle whether to enable TLS in GRPC server |
| testkube-logs.tls.mountClientCACertificate | bool | `false` | If enabled, will also require a client CA certificate to be provided |
| testkube-operator.affinity | object | `{}` | Affinity for Testkube Operator pod assignment. |
| testkube-operator.apiFullname | string | `"testkube-api-server"` | Testkube API full name |
| testkube-operator.apiPort | int | `8088` | Testkube API port |
| testkube-operator.cronJobTemplate | string | `""` |  |
| testkube-operator.agentCronJobs | bool | `true` | Agent cron jobs for scheduling test, suites, workflows |
| testkube-operator.enabled | bool | `true` |  |
| testkube-operator.extraEnvVars | list | `[]` | Extra environment variables to be set on deployment |
| testkube-operator.fullnameOverride | string | `"testkube-operator"` | Testkube Operator fullname override |
| testkube-operator.healthcheckPort | int | `8081` | Testkube Operator healthcheck port |
| testkube-operator.image.digest | string | `""` | Testkube Operator image digest |
| testkube-operator.image.pullPolicy | string | `""` | Testkube Operator image pull policy |
| testkube-operator.image.pullSecrets | list | `[]` | Operator k8s secret for private registries |
| testkube-operator.image.registry | string | `"docker.io"` | Testkube Operator registry |
| testkube-operator.image.repository | string | `"kubeshop/testkube-operator"` | Testkube Operator repository |
| testkube-operator.installCRD | bool | `true` | should the CRDs be installed |
| testkube-operator.livenessProbe.initialDelaySeconds | int | `3` | Initial delay seconds for liveness probe |
| testkube-operator.metricsServiceName | string | `""` | Name of the metrics server. If not specified, default name from the template is used |
| testkube-operator.nameOverride | string | `"testkube-operator"` | Testkube Operator name override |
| testkube-operator.namespace | string | `""` |  |
| testkube-operator.nodeSelector | object | `{}` | Node labels for Testkube Operator pod assignment. |
| testkube-operator.podAnnotations | object | `{}` |  |
| testkube-operator.podLabels | object | `{}` |  |
| testkube-operator.podSecurityContext | object | `{}` | Testkube Operator Pod Security Context |
| testkube-api.postgresql.dsn | string | `"postgres://testkube:postgres5432@testkube-postgresql:5432/backend?sslmode=disable"` | PostgreSQL DSN |
| testkube-api.postgresql.enabled | bool | `false` | use PostgreSQL |
| testkube-operator.preUpgrade.annotations | object | `{}` |  |
| testkube-operator.preUpgrade.enabled | bool | `true` | Upgrade hook is enabled |
| testkube-operator.preUpgrade.image | object | `{"pullPolicy":"IfNotPresent","pullSecrets":[],"registry":"docker.io","repository":"bitnami/kubectl","tag":"1.28.2"}` | Specify image |
| testkube-operator.preUpgrade.labels | object | `{}` |  |
| testkube-operator.preUpgrade.podAnnotations | object | `{}` |  |
| testkube-operator.preUpgrade.podSecurityContext | object | `{}` | Upgrade Pod Security Context |
| testkube-operator.preUpgrade.resources | object | `{}` | Specify resource limits and requests |
| testkube-operator.preUpgrade.securityContext | object | `{}` | Security Context for Upgrade kubectl container |
| testkube-operator.preUpgrade.serviceAccount | object | `{"create":true}` | Create SA for upgrade hook |
| testkube-operator.preUpgrade.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-operator.preUpgrade.ttlSecondsAfterFinished | int | `100` |  |
| testkube-operator.priorityClassName | string | `""` |  |
| testkube-operator.proxy.image.pullSecrets | list | `[]` | Testkube Operator rbac-proxy k8s secret for private registries |
| testkube-operator.proxy.image.registry | string | `"gcr.io"` | Testkube Operator rbac-proxy image registry |
| testkube-operator.proxy.image.repository | string | `"kubebuilder/kube-rbac-proxy"` | Testkube Operator rbac-proxy image repository |
| testkube-operator.proxy.image.tag | string | `"v0.8.0"` | Testkube Operator rbac-proxy image tag |
| testkube-operator.proxy.resources | object | `{}` | Testkube Operator rbac-proxy resource settings |
| testkube-operator.purgeExecutions | bool | `false` | Purge executions on CRD deletion |
| testkube-operator.rbac.create | bool | `true` |  |
| testkube-operator.readinessProbe | object | `{"initialDelaySeconds":3}` | Testkube Operator Readiness Probe parameters |
| testkube-operator.readinessProbe.initialDelaySeconds | int | `3` | Initial delay seconds for readiness probe |
| testkube-operator.replicaCount | int | `1` | Number of Testkube Operator Pod replicas |
| testkube-operator.resources | object | `{}` | Testkube Operator resource settings |
| testkube-operator.securityContext | object | `{"readOnlyRootFilesystem":true}` | Security Context for manager Container |
| testkube-operator.securityContext.readOnlyRootFilesystem | bool | `true` | Make root filesystem of the container read-only |
| testkube-operator.service.port | int | `80` | HTTP Port |
| testkube-operator.service.type | string | `"ClusterIP"` | Adapter service type |
| testkube-operator.serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| testkube-operator.serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| testkube-operator.serviceAccount.name | string | `""` | If not set and create is true, a name is generated using the fullname template |
| testkube-operator.terminationGracePeriodSeconds | int | `10` | Terminating a container that failed its liveness or startup probe after 10s |
| testkube-operator.testConnection | object | `{"enabled":false,"resources":{},"tolerations":[]}` | Test Connection pod |
| testkube-operator.testConnection.resources | object | `{}` | Test Connection resource settings |
| testkube-operator.testConnection.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-operator.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. note: kubebuilder/kube-rbac-proxy:v0.8.0, image used by testkube-operator proxy deployment, doesn't support arm64 nodes |
| testkube-operator.useArgoCDSync | bool | `false` | Use ArgoCD sync owner references |
| testkube-operator.volumes.secret.defaultMode | int | `420` | Testkube Operator webhook certificate volume default mode |
| testkube-operator.webhook.annotations | object | `{}` | Webhook specific annotations |
| testkube-operator.webhook.certificate | object | `{"secretName":"webhook-server-cert"}` | Webhook certificate |
| testkube-operator.webhook.certificate.secretName | string | `"webhook-server-cert"` | Webhook certificate secret name |
| testkube-operator.webhook.createSecretJob.resources | object | `{}` |  |
| testkube-operator.webhook.enabled | bool | `true` | Use webhook |
| testkube-operator.webhook.labels | object | `{}` | Webhook specific labels |
| testkube-operator.webhook.migrate.backoffLimit | int | `1` | Number of retries before considering a Job as failed |
| testkube-operator.webhook.migrate.enabled | bool | `true` | Deploy Migrate Job |
| testkube-operator.webhook.migrate.image.pullPolicy | string | `"IfNotPresent"` | Migrate container job image pull policy |
| testkube-operator.webhook.migrate.image.pullSecrets | list | `[]` | Migrate container job k8s secret for private registries |
| testkube-operator.webhook.migrate.image.registry | string | `"docker.io"` | Migrate container job image registry |
| testkube-operator.webhook.migrate.image.repository | string | `"rancher/kubectl"` | Migrate container job image name |
| testkube-operator.webhook.migrate.image.version | string | `"v1.23.7"` | Migrate container job image tag |
| testkube-operator.webhook.migrate.resources | object | `{}` | Migrate job resources settings |
| testkube-operator.webhook.migrate.securityContext | object | `{"readOnlyRootFilesystem":true}` | Security Context for webhook migrate Container |
| testkube-operator.webhook.migrate.securityContext.readOnlyRootFilesystem | bool | `true` | Make root filesystem of the container read-only |
| testkube-operator.webhook.migrate.ttlSecondsAfterFinished | int | `100` |  |
| testkube-operator.webhook.name | string | `"testkube-operator-webhook-admission"` | Name of the webhook |
| testkube-operator.webhook.namespaceSelector | object | `{}` |  |
| testkube-operator.webhook.patch.annotations | object | `{}` | Annotations to add to the patch Job |
| testkube-operator.webhook.patch.createSecretJob.resources | object | `{}` | kube-webhook-certgen create secret Job resource settings |
| testkube-operator.webhook.patch.createSecretJob.securityContext | object | `{"readOnlyRootFilesystem":true}` | Security Context for webhook create container |
| testkube-operator.webhook.patch.createSecretJob.securityContext.readOnlyRootFilesystem | bool | `true` | Make root filesystem of the container read-only |
| testkube-operator.webhook.patch.enabled | bool | `true` |  |
| testkube-operator.webhook.patch.image.pullPolicy | string | `"Always"` | patch job image pull policy |
| testkube-operator.webhook.patch.image.pullSecrets | list | `[]` | patch job k8s secret for private registries |
| testkube-operator.webhook.patch.image.registry | string | `"docker.io"` | patch job image registry |
| testkube-operator.webhook.patch.image.repository | string | `"dpejcev/kube-webhook-certgen"` | patch job image name |
| testkube-operator.webhook.patch.image.version | string | `"1.0.11"` | patch job image tag |
| testkube-operator.webhook.patch.labels | object | `{}` | Pod specific labels |
| testkube-operator.webhook.patch.nodeSelector | object | `{}` | Node labels for pod assignment |
| testkube-operator.webhook.patch.patchWebhookJob.resources | object | `{}` | kube-webhook-certgen patch webhook Job resource settings |
| testkube-operator.webhook.patch.patchWebhookJob.securityContext | object | `{"readOnlyRootFilesystem":true}` | Security Context for webhook patch container |
| testkube-operator.webhook.patch.patchWebhookJob.securityContext.readOnlyRootFilesystem | bool | `true` | Make root filesystem of the container read-only |
| testkube-operator.webhook.patch.podAnnotations | object | `{}` | Pod annotations to add to the patch Job |
| testkube-operator.webhook.patch.podSecurityContext | object | `{}` | kube-webhook-certgen Job Security Context |
| testkube-operator.webhook.patch.serviceAccount.annotations | object | `{}` | SA specific annotations |
| testkube-operator.webhook.patch.serviceAccount.name | string | `"testkube-operator-webhook-cert-mgr"` | SA name |
| testkube-operator.webhook.patch.tolerations | list | `[]` | Tolerations to schedule a workload to nodes with any architecture type. Required for deployment to GKE cluster. |
| testkube-operator.webhook.patch.ttlSecondsAfterFinished | int | `100` |  |
| testkube-operator.webhook.patchWebhookJob.resources | object | `{}` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.13.1](https://github.com/norwoodj/helm-docs/releases/v1.13.1)

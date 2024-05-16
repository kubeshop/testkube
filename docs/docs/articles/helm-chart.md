# Testkube Helm Charts

## 1. Add the Kubeshop Helm repository.

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts
```

If this repo already exists, run `helm repo update` to retrieve
the `latest` versions of the packages. You can then run `helm search repo
testkube` to see the charts.

## 2. Install the `testkube` chart.

```sh
helm install --create-namespace my-testkube kubeshop/testkube
```

:::note
By default, the namespace for the installation will be `testkube`. If the `testkube` namespace does not exist, it will be created for you.

If you wish to install into a different namespace, please use following command:

```sh
helm install --namespace namespace_name my-testkube kubeshop/testkube
```

To uninstall the `testkube` chart if it was installed into the default namespace:

```sh
helm delete my-testkube kubeshop/testkube
```

And from a namespace other than `testkube`:

```sh
helm delete --namespace namespace_name my-testkube kubeshop/testkube
```

:::

## Testkube Multi-namespace Feature

It is possible to deploy multiple Testkube instances into the same Kubernetes cluster. Please follow these installation commands.

**1. For new installations:**

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts

helm install testkube kubeshop/testkube --namespace testkube --create-namespace --set testkube-api.multinamespace.enabled=true

helm install testkube1 kubeshop/testkube -n testkube1 --create-namespace --set testkube-api.multinamespace.enabled=true --set testkube-operator.enabled=false
```

These commands will deploy Testkube components into two namespaces: testkube and testkube1 and will create a watcher role to watch k8s resources in each namespace respectively. If you need to watch resources besides the installation namespace, please add them to the **_additionalNamespaces_** variable in **_testkube-api_** section:

```sh
testkube-api:
  additionalNamespaces:
  - namespace2
  - namespace3

```

Additionally, It is possible to change the namespace for **_testkube-operator_** by setting a value for **_namespace_** variable in the **_testkube-operator_** section:

```sh
testkube-operator:
  namespace: testkube-system
```

:::note

Please note that the **Testkube Operator** creates **ClusterRoles**, so for the second deployment of Testkube, we need to disable the Operator, because it will fail with a `resources already exist` error. Be aware that the Operator is deployed once with the first chart installation of Testkube. Therefore, if you uninstall the first release, it will uninstall the Operator as well.

:::
**2. For the users who already have Testkube installed**

Run the following commands to deploy Testkube in another namespace:

```sh
helm repo update kubeshop

helm install testkube1 kubeshop/testkube --namespace testkube1 --set testkube-api.multinamespace.enabled=true --set testkube-operator.enabled=false
```

#### Helm Properties

The following Helm defaults are used in the `testkube` chart:

| Parameter                              | Is optional | Default                              | Additional details                          |
| -------------------------------------- | ----------- | ------------------------------------ | ------------------------------------------- |
| mongodb.auth.enabled                   | yes         | false                                |
| mongodb.service.port                   | yes         | "27017"                              |
| mongodb.service.portName               | yes         | "mongodb"                            |
| mongodb.service.nodePort               | yes         | true                                 |
| mongodb.service.clusterIP              | yes         | ""                                   |
| mongodb.nameOverride                   | yes         | "mongodb"                            |
| mongodb.fullnameOverride               | yes         | "testkube-mongodb"                   |
| testkube-api.image.repository          | yes         | "kubeshop/testkube-api-server"       |
| testkube-api.image.pullPolicy          | yes         | "Always"                             |
| testkube-api.image.tag                 | yes         | "latest"                             |
| testkube-api.service.type              | yes         | "NodePort"                           |
| testkube-api.service.port              | yes         | 8088                                 |
| testkube-api.mongodb.dsn               | yes         | "mongodb://testkube-mongodb:27017"   |
| testkube-api.nats.uri                  | yes         | "nats://testkube-nats"               |
| testkube-api.telemetryEnabled          | yes         | true                                 |
| testkube-api.storage.endpoint          | yes         | testkube-minio-service-testkube:9000 |
| testkube-api.storage.accessKeyId       | yes         | minio                                |
| testkube-api.storage.accessKey         | yes         | minio123                             |
| testkube-api.storage.scrapperEnabled   | yes         | true                                 |
| testkube-api.slackToken                | yes         | ""                                   |
| testkube-api.slackSecret               | yes         | ""                                   |
| testkube-api.slackConfig               | yes         | ""                                   |
| testkube-api.jobServiceAccountName     | yes         | ""                                   |
| testkube-api.logs.storage              | no          | "minio"                              |
| testkube-api.logs.bucket               | no          | "testkube-logs"                      |
| testkube-api.cdeventsTarget            | yes         | ""                                   |
| testkube-api.dashboardUri              | yes         | ""                                   |
| testkube-api.clusterName               | yes         | ""                                   |
| testkube-api.storage.compressArtifacts | yes         | true                                 |
| testkube-api.enableSecretsEndpoint     | yes         | false                                | [Learn more](./secrets-enable-endpoint.md)  |
| testkube-api.disableMongoMigrations    | yes         | false                                |
| testkube-api.enabledExecutors          | yes         | ""                                   |
| testkube-api.disableSecretCreation     | yes         | false                                | [Learn more](./secrets-disable-creation.md) |
| testkube-api.defaultStorageClassName   | yes         | ""                                   |
| testkube-api.enableK8sEvents           | yes         | true                                 |

> For more configuration parameters of a `MongoDB` chart please visit:
> <https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters>

> For more configuration parameters of an `NATS` chart please visit:
> <https://docs.nats.io/running-a-nats-service/nats-kubernetes/helm-charts>

:::note

Please note that we use **global** parameters in our `values.yaml`:

```
global:
  imageRegistry: ""
  imagePullSecrets: []
  labels: {}
  annotations: {}
```

They override all sub-chart values for the image parameters if specified.

:::

# Install Testkube OSS

Welcome to the Open Source version of Testkube!

Testkube OSS includes our full test execution and orchestration engine. It is deployed as a standalone agent.

Designed to integrate seamlessly with your Kubernetes clusters, Testkube offers flexibility and power. For those searching for a quicker and streamlined experience, we suggest [signing up for Testkube Pro](https://app.testkube.io/). However, for those that prefer to dip their toes in the water before diving into the deep, then the open source version is the better choice.

Visit [Open Source or Pro][oss-vs-pro] to see a comparison of features to help you with your choice. Also, take a look at our [Quick Setup Video](https://www.youtube.com/watch?v=ynzEkOUhxKk).

You can install with the CLI or Helm. The following components will be installed into your Kubernetes cluster:

- Create a Testkube namespace.
- Deploy the Testkube API.
- Use MongoDB for test results and Minio for artifact storage (optional; disable with --no-minio).
- Testkube will listen and manage all the CRDs for Tests, TestSuites, Executors, etc… inside the Testkube namespace.

Once installed you can verify your installation and check that Testkube is up and running with
`kubectl get all -n testkube`. Once validated, you're ready to unleash the full potential of Testkube in your environment. Testkube OSS is here to help you to powering your development and testing workflows seamlessly.

## Minimum Resource Requirements

To ensure optimal performance, the initial setup requires a minimum of **2 CPU cores** and **8 GB of RAM**. This configuration is suitable for basic operations.

For environments with higher demands or fluctuating workloads, we recommend implementing an **autoscaler**. This allows for dynamic scaling of resources based on actual usage, ensuring both efficient performance and cost-effectiveness. Users can configure the autoscaler according to their specific needs, adapting to varying workloads seamlessly.

## Install with CLI

You can install the standalone agent by executing the following command.
By default it will install within the `testkube` namespace for your
current Kubernetes context.

```
testkube init
```

## Install with Helm

### 1. Add the Kubeshop Helm repository.

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts
```

If this repo already exists, run `helm repo update` to retrieve
the `latest` versions of the packages. You can then run `helm search repo
testkube` to see the charts.

### 2. Install the `testkube` chart.

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

#### Helm Properties

The following Helm defaults are used in the `testkube` chart:

| Parameter                              | Is optional | Default                              | Additional details             |
| -------------------------------------- | ----------- | ------------------------------------ | ------------------------------ |
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
| testkube-api.enableSecretsEndpoint     | yes         | false                                | [Learn more][secrets-endpoint] |
| testkube-api.disableMongoMigrations    | yes         | false                                |
| testkube-api.enabledExecutors          | yes         | ""                                   |
| testkube-api.disableSecretCreation     | yes         | false                                | [Learn more][secrets-creation] |
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

## Upgrade Testkube OSS

See [upgrade][upgrade] for instructions on how to upgrade your Testkube Core Open Source components.

[secrets-endpoint]: /articles/secrets-enable-endpoint
[secrets-creation]: /articles/secrets-disable-creation
[oss-vs-pro]: /articles/open-source-or-pro
[upgrade]: /articles/upgrade

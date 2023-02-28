# Testkube Helm Charts

## 1. Add the Kubeshop Helm repository

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts
```

If this repo already exists, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
testkube` to see the charts.

## 2. Install the `testkube` chart

```sh
helm install --create-namespace my-testkube testkube/testkube
```

:::note
By default, the namespace for the installation will be `testkube`. If the `testkube` namespace does not exist, it will be created for you.

If you wish to install into a different namespace, please use following command:

```sh
helm install --namespace namespace_name my-testkube testkube/testkube
```

To uninstall the `testkube` chart if it was installed into default namespace:

```sh
helm delete my-testkube testkube/testkube
```

And from a namespace other than `testkube`:

```sh
helm delete --namespace namespace_name my-testkube testkube/testkube
```
:::

#### Helm Properties

The following Helm defaults are used in the `testkube` chart:

| Parameter                            | Is optional | Default                              |
| ------------------------------------ | ----------- | ------------------------------------ |
| mongodb.auth.enabled                 | yes         | false                                |
| mongodb.service.port                 | yes         | "27017"                              |
| mongodb.service.portName             | yes         | "mongodb"                            |
| mongodb.service.nodePort             | yes         | true                                 |
| mongodb.service.clusterIP            | yes         | ""                                   |
| mongodb.nameOverride                 | yes         | "mongodb"                            |
| mongodb.fullnameOverride             | yes         | "testkube-mongodb"                   |
| testkube-api.image.repository        | yes         | "kubeshop/testkube-api-server"       |
| testkube-api.image.pullPolicy        | yes         | "Always"                             |
| testkube-api.image.tag               | yes         | "latest"                             |
| testkube-api.service.type            | yes         | "NodePort"                           |
| testkube-api.service.port            | yes         | 8088                                 |
| testkube-api.mongodb.dsn             | yes         | "mongodb://testkube-mongodb:27017"   |
| testkube-api.nats.uri                | yes         | "nats://testkube-nats"               |
| testkube-api.telemetryEnabled        | yes         | true                                 |
| testkube-api.storage.endpoint        | yes         | testkube-minio-service-testkube:9000 |
| testkube-api.storage.accessKeyId     | yes         | minio                                |
| testkube-api.storage.accessKey       | yes         | minio123                             |
| testkube-api.storage.scrapperEnabled | yes         | true                                 |
| testkube-api.slackToken              | yes         | ""                                   |
| testkube-api.slackTemplate           | yes         | ""                                   |
| testkube-api.slackConfig             | yes         | ""                                   |
| testkube-api.jobServiceAccountName   | yes         | ""                                   |
| testkube-api.logs.storage            | no          | "minio"                              |
| testkube-api.logs.bucket             | no          | "testkube-logs"                      |

>For more configuration parameters of `MongoDB` chart please visit:
<https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters>

>For more configuration parameters of `NATS` chart please visit:
<https://docs.nats.io/running-a-nats-service/nats-kubernetes/helm-charts>

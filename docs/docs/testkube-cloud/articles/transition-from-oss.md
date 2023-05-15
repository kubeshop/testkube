# Transition from Testkube OSS

To migrate Testkube OSS to Cloud you need to install Testkube in Cloud Agent mode. Testkube Cloud Agent is the Testkube engine for managing test runs into your cluster. It sends data to Testkube's Cloud Servers. Its main responsibility is to manage test workloads and to get insight into Testkube resources stored in the cluster.


## Installing the Agent

Please follow the [install steps](installing-agent.md) to get started using the Testkube Agent.

You will copy the Helm command to install Testkube in your cluster:

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts
helm repo update
helm upgrade \
  --install \
  --create-namespace testkube kubeshop/testkube \
  --set mongodb.enabled=false \
  --namespace testkube \
  --set testkube-api.minio.enabled=false \
  --set testkube-api.cloud.key=tkcagnt_YOUR_TOKEN
```

:::danger

Please keep in mind that the default install will REMOVE existing MongoDB, MinIO and Dashboard pods!

To keep the pods, set the below options to true (3 values for MongoDB, MinIO, Dashboard):

```sh
 --set testkube-api.minio.enabled=true --set mongodb.enabled=true --set testkube-dashboard.enabled=true
```

:::

## Setting the Testkube CLI Context to Agent Mode

Please follow the [context management guide](managing-cli-context.md) to configure your Testkube CLI in Cloud mode.

## Migrating the Testkube Resources

Currently there is no automatic migration tool for existing Testkube OSS resources. This is planned for coming releases.

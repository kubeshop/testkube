# Transition from Testkube OSS

To migrate Testkube OSS to Cloud you need to install Testkube in Cloud Agent mode. Testkube Cloud Agent is the Testkube engine for managing test runs into your cluster. It sends data to Testkubes Cloud Servers. It's main responsibility is to manage test workloads and to get insight into Testkube resources stored in the cluster.


## Installing the Agent

Please follow the [install steps](installing-agent.md) to get started using the Testkube Agent.

```
helm repo add kubeshop https://kubeshop.github.io/helm-charts ; helm repo update && helm upgrade --install --create-namespace testkube kubeshop/testkube --set testkube-api.cloud.key=tkcagnt_aaaaaaaaaaaaaaaaaaaaakey --set testkube-api.minio.enabled=false --set mongodb.enabled=false --namespace testkube
```

WARNING! Please keep in mind that default install will REMOVE existing MongoDB, Minio and Dashboard pods!

To keep them set below options to true (3 values for MongoDB, MinIO, Dashboard):
```sh
 --set testkube-api.minio.enabled=true --set mongodb.enabled=true --set testkube-dashboard.enabled=true
```

## Setting the Testkube CLI context to the agent mode

Please follow the [context management](managing-cli-context.md) to configure your Testkube CLI in Cloud mode.


## Migrating the Testkube Resources

Currently there is no automatic migration tool for existing Testkube OSS resources. But we have plan for it in incoming releases.

# Installation 

To get TestKube up and running you will need to

1. Install the kubectl testkube plugin
2. Install TestKube in your cluster 
3. Configure TestKube's Dashboard UI Ingress for your ingress-controller if needed.

## Install the kubectl testkube plugin

To install on Linux or MacOs run 
```sh
bash < <(curl -sSLf https://kubeshop.github.io/testkube/install.sh )
```

For Windows download desired binary from https://github.com/kubeshop/testkube/releases, unpack the binary and add it to `%PATH%`. 

We have plans to build installers for most popular OS and system distros.

## Install `testkube` components in your cluster

The testkube kubectl plugin provides an install command to install testkube in your cluster. Internally 
this uses Helm and so you will need to have recent `helm` command installed on your system.

Run 
```shell
kubectl testkube install
```

You should have everything installed 🏅

By default testkube is installed in `testkube` namespace but you can change it in manual install if you want.

If you want testkube to provide the endpoint for the kubest dashboard use `kubectl testkube install -i` with the `-i` or `--ingress` option, it will setup a ingress-nginx controller for you in a managed cluster(for baremetal clusters this should be set up manually before installing testkube).
## Uninstall `testkube`

You can uninstall TestKube using uninstall command integrated into testkube plugin. 

```
kubectl testkube uninstall [--remove-crds]
```

Optionally you can use `--remove-crds` flag which clean all installed Custom Resource Definitions installed by TestKube.


### Manual testkube Helm charts installation

[Helm](https://helm.sh) must be installed to use the charts.  
Please refer to  Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the Kubeshop Helm repository  as follows:

```sh
helm repo add testkube https://kubeshop.github.io/helm-charts
```

If you had already added this repo earlier, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
testkube` to see the charts.

To install the `testkube` chart:

```sh
helm install my-testkube testkube/testkube
```

To uninstall the `testkube` chart:

```sh
helm delete my-testkube testkube/testkube
```

### Helm Properties

Helm defaults used in the `testkube` chart:

| Parameter | Is optional | Default |
| --- | --- | --- |
| mongodb.auth.enabled | yes | false |
| mongodb.service.port | yes | "27017" |
| mongodb.service.portNmae | yes | "mongodb" |
| mongodb.service.nodePort | yes | true |
| mongodb.service.clusterIP | yes | "" |
| mongodb.nameOverride | yes | "mongodb" |
| mongodb.fullnameOverride | yes | "testkube-mongodb" |
| api-server.image.repository | yes | "kubeshop/testkube-api-server" |
| api-server.image.pullPolicy | yes | "Always" |
| api-server.image.tag | yes | "latest" |
| api-server.service.type | yes | "NodePort" |
| api-server.service.port | yes | 8088 |
| api-server.mongoDSN | yes | "mongodb://testkube-mongodb:27017" |
| api-server.telemetryDisabled | yes | false |

>For more configuration parameters of `MongoDB` chart please look here:
https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters

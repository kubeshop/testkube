# Installation 

To get TestKube up and running you will need to

1. Install the kubectl testkube plugin
2. Install TestKube in your cluster 
3. Configure TestKube's Dashboard UI Ingress for your ingress-controller if needed.

## Install the kubectl testkube plugin

### Installing on MacOS 

We're building Homebew tap for each release, so you can easily install TestKube with Homebrew.

```sh
brew tap kubeshop/homebrew-testkube
brew install kubeshop/testkube
```

### Installing on Linux or MaxOS with install script

To install on Linux or MacOs run 
```sh
bash < <(curl -sSLf https://kubeshop.github.io/testkube/install.sh )
```

### Alternative installation method (manual)

If you don't like automatic scripts you can always use manuall install:

1. Download binary with version of your choice (recent one is recommended)
2. Upack it (tar -zxvf testkube_0.6.5_Linux_arm64.tar.gz)
3. Move it to location in the path for example `mv  testkube_0.6.5_Linux_arm64/kubectl-testkube /usr/local/bin/kubectl-testkube`

For Windows download desired binary from https://github.com/kubeshop/testkube/releases, unpack the binary and add it to `%PATH%`. 

We have plans to build installers for most popular OS and system distros [#161](https://github.com/kubeshop/testkube/issues/161).

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
helm install --create-namespace my-testkube testkube/testkube
```
Please note that by default it will be looking for the   `testkube` namespace to be installed into. And if doesn't find it the namespace will be created for you.

If you wish to install it into a different namespace please use following command instead:
```sh
helm install --namespace namespace_name my-testkube testkube/testkube
```


To uninstall the `testkube` chart if it was installed into default namespace:

```sh
helm delete my-testkube testkube/testkube
```
And from different than `testkube` namespace:
```sh
helm delete --namespace namespace_name my-testkube testkube/testkube
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
| api-server.storage.endpoint_port | yes | 9000 |
| api-server.storage.accessKeyId | yes | minio |
| api-server.storage.accessKey | yes | minio123 |
| api-server.storage.scrapperEnabled | yes | false |

>For more configuration parameters of `MongoDB` chart please look here:
https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters

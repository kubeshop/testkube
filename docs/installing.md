# Installation 

To get Kubtest up and running you will need to

1. Install the kubectl kubtest plugin
2. Install Kubtest in your cluster 

## Install the kubectl kubtest plugin

To install on Linux or MacOs run 
```sh
$ curl -sSLf https://kubeshop.github.io/kubtest/install.sh | sudo bash
```

For Windows download desired binary from https://github.com/kubeshop/kubtest/releases, unpack the binary and add it to `%PATH%`. 

We have plans to build installers for most popular OS and system distros.

#### MacOS 

To run kubectl-kubtest you need to remove quarantine flags from file

```sh
xattr -d com.apple.quarantine kubectl-kubtest
```

## Install kubtest components in your cluster

The kubtest kubectl plugin provides an install command to install kubtest in your cluster. Internally 
this uses Helm and so you will need to have recent `helm` command installed on your system.

Run 
```shell
kubectl kubtest install
```

You should have everything installed 🏅

By default kubtest is installed in `default` namespace but you can change it in manual install if you want.

If you want kubtest to provide the endpoint for the kubest dashboard use `kubectl kubtest install -i` with the `-i` or `--ingress` option, it will setup a ingress-nginx controller for you in a managed cluster(for baremetal clusters this should be set up manually before installing kubtest).

### Manual kubtest Helm charts installation

[Helm](https://helm.sh) must be installed to use the charts.  
Please refer to  Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the Kubeshop Helm repository  as follows:

```sh
helm repo add kubtest https://kubeshop.github.io/helm-charts
```

If you had already added this repo earlier, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
kubtest` to see the charts.

To install the `kubtest` chart:

```sh
helm install my-kubtest kubtest/kubtest
```

To uninstall the `kubtest` chart:

```sh
helm delete my-kubtest kubtest/kubtest
```

### Helm Properties

Helm defaults used in the `kubtest` chart:

| Parameter | Is optional | Default |
| --- | --- | --- |
| mongodb.auth.enabled | yes | false |
| mongodb.service.port | yes | "27017" |
| mongodb.service.portNmae | yes | "mongodb" |
| mongodb.service.nodePort | yes | true |
| mongodb.service.clusterIP | yes | "" |
| mongodb.nameOverride | yes | "mongodb" |
| mongodb.fullnameOverride | yes | "kubtest-mongodb" |
| api-server.image.repository | yes | "kubeshop/kubtest-api-server" |
| api-server.image.pullPolicy | yes | "Always" |
| api-server.image.tag | yes | "latest" |
| api-server.service.type | yes | "NodePort" |
| api-server.service.port | yes | 8088 |
| api-server.mongoDSN | yes | "mongodb://kubtest-mongodb:27017" |
| api-server.postmanExecutorURI | yes | "http://kubtest-postman-executor:8082" |
| postman-executor.image.repository | yes | "kubeshop/kubtest-postman-executor" |
| postman-executor.image.pullPolicy | yes | "Always" |
| postman-executor.image.tag | yes | "latest" |
| postman-executor.service.type | yes | "NodePort" |
| postman-executor.service.port | yes | 8082 |
| postman-executor.mongoDSN | yes | "mongodb://kubtest-mongodb:27017" |
| postman-executor.apiServerURI | yes | "http://kubtest-api-server:8088" |

>For more configuration parameters of `MongoDB` chart please look here:
https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters

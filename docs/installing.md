# Installation 

## kubectl-kubtest plugin

To install `kubectl kubtest` plugin please download [latest release of kubtest](
https://github.com/kubeshop/kubtest/releases) unpack binary and put it somewhere in 
your `$PATH`. 

We have plans to build installers for most popular OS and system distros.

### MacOS 

to run kubectl-kubtest you need to remove quarantine flags from file

```sh
xattr -d com.apple.quarantine kubectl-kubtest
```


## Cluster

For installation we're using Helm charts so you need to have recent `helm` command installed
on your system. 


### kubtest cluster install from plugin

To simplify install you can use following command to install all required components of kubtest: 

```
kubectl kubtest install
```

You should have everything installed 🏅

By default kubtest is installed in `default` namespace but you can change it in manual install if you want.


### Manual kubtest Helm charts installation

Helm install 

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:
```sh
helm repo add kubtest https://kubeshop.github.io/helm-charts
```
If you had already added this repo earlier, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
kubtest` to see the charts.

To install the `kubtest` chart:
```sh
helm install my-<chart-name> kubtest/kubtest
```
To uninstall the `kubtest` chart:
```sh
helm delete my-<chart-name> kubtest/kubtest
```

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
| api-server.service.port | yes | 8080 |
| api-server.mongoDSN | yes | "mongodb://kubtest-mongodb:27017" |
| api-server.postmanExecutorURI | yes | "http://kubtest-postman-executor:8082" |
| postman-executor.image.repository | yes | "kubeshop/kubtest-postman-executor" |
| postman-executor.image.pullPolicy | yes | "Always" |
| postman-executor.image.tag | yes | "latest" |
| postman-executor.service.type | yes | "NodePort" |
| postman-executor.service.port | yes | 8082 |
| postman-executor.mongoDSN | yes | "mongodb://kubtest-mongodb:27017" |
| postman-executor.apiServerURI | yes | "http://kubtest-api-server:8080" |

>For more configuration parameters of `MongoDB` chart please look here:
https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters

# Installation 

For installation we're using Helm charts. To simplify install 
you can use following command to install all required components of KubeTest: 

```
kubectl kubetest install
```

## Helm installation

Helm install 

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:
```sh
helm repo add kubetest https://kubeshop.github.io/kubetest
```
If you had already added this repo earlier, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
kubetest` to see the charts.

To install the `kubetest` chart:
```sh
helm install my-<chart-name> kubetest/kubetest
```
To uninstall the `kubetest` chart:
```sh
helm delete my-<chart-name> kubetest/kubetest
```

Helm defaults used in the `Kubetest` chart:

| Parameter | Is optional | Default |
| --- | --- | --- |
| mongodb.auth.enabled | yes | false |
| mongodb.service.port | yes | "27017" |
| mongodb.service.portNmae | yes | "mongodb" |
| mongodb.service.nodePort | yes | true |
| mongodb.service.clusterIP | yes | "" |
| mongodb.nameOverride | yes | "mongodb" |
| mongodb.fullnameOverride | yes | "kubetest-mongodb" |
| api-server.image.repository | yes | "kubeshop/kubetest-api-server" |
| api-server.image.pullPolicy | yes | "Always" |
| api-server.image.tag | yes | "latest" |
| api-server.service.type | yes | "NodePort" |
| api-server.service.port | yes | 8080 |
| api-server.mongoDSN | yes | "mongodb://kubetest-mongodb:27017" |
| api-server.postmanExecutorURI | yes | "http://kubetest-postman-executor:8082" |
| postman-executor.image.repository | yes | "kubeshop/kubetest-postman-executor" |
| postman-executor.image.pullPolicy | yes | "Always" |
| postman-executor.image.tag | yes | "latest" |
| postman-executor.service.type | yes | "NodePort" |
| postman-executor.service.port | yes | 8082 |
| postman-executor.mongoDSN | yes | "mongodb://kubetest-mongodb:27017" |
| postman-executor.apiServerURI | yes | "http://kubetest-api-server:8080" |

>For more configuration parameters of `MongoDB` chart please look here:
https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters

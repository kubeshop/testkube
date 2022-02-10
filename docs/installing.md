# Installation

To get Testkube up and running you will need to

1. Install the kubectl testkube plugin
2. Install Testkube in your cluster
3. Configure Testkube's Dashboard UI Ingress for your ingress-controller if needed.

## Install the kubectl testkube plugin

### Installing on MacOS

We're building Homebew tap for each release, so you can easily install Testkube with Homebrew.

```sh
brew tap kubeshop/homebrew-testkube
brew install kubeshop/testkube/testkube
```

If you want to upgrade testkube please run following command

```sh
brew update 
brew upgrade testkube
```

### Installing on Linux or MacOS with install script

To install on Linux or MacOs run

```sh
bash < <(curl -sSLf https://kubeshop.github.io/testkube/install.sh )
```

### Alternative installation method (manual)

If you don't like automatic scripts you can always use manuall install:

1. Download binary with version of your choice (recent one is recommended)
2. Upack it (tar -zxvf testkube_0.6.5_Linux_arm64.tar.gz)
3. Move it to a location in the PATH. For example `mv  testkube_0.6.5_Linux_arm64/kubectl-testkube /usr/local/bin/kubectl-testkube`

For Windows, download the binary from [here](https://github.com/kubeshop/testkube/releases), unpack the binary and add it to `%PATH%`.

We have plans to build installers for the most popular Operating Systems and system distros [#161](https://github.com/kubeshop/testkube/issues/161).

## Install `testkube` components in your cluster

The testkube kubectl plugin provides an install command to install testkube in your cluster.

Run

```shell
kubectl testkube install
```

The above command will install the following components in your Kubernetes cluster:

1. Testkube API
2. `testkube` namespace
3. CRD for scripts
4. MongoDB
5. Minio - default (can be disabled with `--no-minio` flag if you want to use S3 buckets)
6. Dashboard - default (can be disabled with `--no-dasboard` flag)
7. Jetstack certificate manager for `testkube` namespace (can be disabled with `--no-jetstack` flag). It will not be installed, if it's already installed to your Kubernetes cluster

You can confirm it by running:

```sh
kubectl get all -n testkube
```

Output:

```sh
NAME                                           READY   STATUS    RESTARTS   AGE
pod/cert-manager-847544bbd-fw2h8               1/1     Running   0          4m51s
pod/cert-manager-cainjector-5c747645bf-qgftx   1/1     Running   0          4m51s
pod/cert-manager-webhook-77b946cb6d-dl6gb      1/1     Running   0          4m51s
pod/testkube-dashboard-748cbcbb66-q8zzp        1/1     Running   0          4m51s
pod/testkube-api-server-546777c9f7-7g4kg       1/1     Running   0          4m51s
pod/testkube-mongodb-5d95f44fdd-cxqz6          1/1     Running   0          4m51s
pod/testkube-minio-testkube-64cd475b94-562hz   1/1     Running   0          4m51s

NAME                                      TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                                        AGE
service/cert-manager                      ClusterIP   10.106.81.214   <none>        9402/TCP                                       2d20h
service/cert-manager-webhook              ClusterIP   10.104.228.254  <none>        443/TCP                                        2d20h
service/testkube-minio-service-testkube   NodePort    10.43.121.107   <none>        9000:31222/TCP,9090:32002/TCP,9443:32586/TCP   4m51s
service/testkube-api-server               NodePort    10.43.66.13     <none>        8088:32203/TCP                                 4m51s
service/testkube-mongodb                  ClusterIP   10.43.126.230   <none>        27017/TCP                                      4m51s
service/testkube-dashboard                NodePort    10.43.136.34    <none>        80:31991/TCP                                   4m51s

NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/cert-manager              1/1     1            1           4m51s
deployment.apps/cert-manager-cainjector   1/1     1            1           4m51s
deployment.apps/cert-manager-webhook      1/1     1            1           4m51s
deployment.apps/testkube-dashboard        1/1     1            1           4m51s
deployment.apps/testkube-api-server       1/1     1            1           4m51s
deployment.apps/testkube-mongodb          1/1     1            1           4m51s
deployment.apps/testkube-minio-testkube   1/1     1            1           4m51s

NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/cert-manager-847544bbd               1         1         1       4m51s
replicaset.apps/cert-manager-cainjector-5c747645bf   1         1         1       4m51s
replicaset.apps/cert-manager-webhook-77b946cb6d      1         1         1       4m51s
replicaset.apps/testkube-dashboard-748cbcbb66        1         1         1       4m51s
replicaset.apps/testkube-api-server-546777c9f7       1         1         1       4m51s
replicaset.apps/testkube-mongodb-5d95f44fdd          1         1         1       4m51s
replicaset.apps/testkube-minio-testkube-64cd475b94   1         1         1       4m51s
```

By default testkube is installed in the `testkube` namespace.

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

| Parameter                          | Is optional | Default                              |
| ---------------------------------- | ----------- | ------------------------------------ |
| mongodb.auth.enabled               | yes         | false                                |
| mongodb.service.port               | yes         | "27017"                              |
| mongodb.service.portNmae           | yes         | "mongodb"                            |
| mongodb.service.nodePort           | yes         | true                                 |
| mongodb.service.clusterIP          | yes         | ""                                   |
| mongodb.nameOverride               | yes         | "mongodb"                            |
| mongodb.fullnameOverride           | yes         | "testkube-mongodb"                   |
| api-server.image.repository        | yes         | "kubeshop/testkube-api-server"       |
| api-server.image.pullPolicy        | yes         | "Always"                             |
| api-server.image.tag               | yes         | "latest"                             |
| api-server.service.type            | yes         | "NodePort"                           |
| api-server.service.port            | yes         | 8088                                 |
| api-server.mongoDSN                | yes         | "mongodb://testkube-mongodb:27017"   |
| api-server.telemetryDisabled       | yes         | false                                |
| api-server.storage.endpoint        | yes         | testkube-minio-service-testkube:9000 |
| api-server.storage.accessKeyId     | yes         | minio                                |
| api-server.storage.accessKey       | yes         | minio123                             |
| api-server.storage.scrapperEnabled | yes         | true                                 |

>For more configuration parameters of `MongoDB` chart please look here:
<https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters>

## Uninstall `testkube`

You can uninstall Testkube using the uninstall command integrated into the testkube plugin.

```sh
kubectl testkube uninstall [--remove-crds]
```

Optionally you can use the `--remove-crds` flag which will clean all installed Custom Resource Definitions installed by Testkube.

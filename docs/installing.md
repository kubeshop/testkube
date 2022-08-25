# Installation Steps

To get Testkube up and running you need to:

1. Install the Testkube CLI.
2. Use HELM or the Testkube CLI to to install Testkube Server components in your cluster.
3. (optional) Configure Testkube's Dashboard UI Ingress for your ingress-controller, if needed.

Watch the full installation video from our product experts: [Testkube Installation Video](https://www.youtube.com/watch?v=bjQboi3Etys).

# **1. Testkube CLI**
Package dependencies:
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)

## From Scripts
To install on Linux or MacOs, run
```sh
bash < <(curl -sSLf https://kubeshop.github.io/testkube/install.sh )
```
## Through Package Managers
### **Homebrew (MacOS)**

You can install Testkube from Homebrew:
```sh
brew install testkube
```
Or directly from our tap. The Homebrew mantainers take a few days/or week to approve each one of our releases so you can use our tap to make sure you always have the most recent release.
```sh
brew tap kubeshop/homebrew-testkube
brew install kubeshop/testkube/testkube
```


### **Chocolatey (Windows)**

You can use [Chocolatey](https://chocolatey.org/install) to install  version from our own chocolatey repository .

```sh
choco source add --name=testkube_repo --source=http://chocolatey.testkube.io/chocolatey
choco install testkube
```

#### **APT (Debian/Ubuntu)**

1. Download our public GPG key, and add them to the trusted keys:
```sh
wget -qO - https://repo.testkube.io/key.pub | sudo apt-key add -
```
2. Add our repository to your apt sources:
```sh
echo "deb https://repo.testkube.io/linux linux main" | sudo tee -a /etc/apt/sources.list
```
3. Make sure to get the updates:
```sh
sudo apt-get update
```

4. Install Testkube:
```sh
sudo apt-get install -y testkube
```

## **Manual Download**

If you don't want automatic scripts or package managers you can always do a manual install:

1. Download binary with the version of your choice and platform of your choice [here](https://github.com/kubeshop/testkube/releases)
2. Unpack it. For example, (tar -zxvf testkube_0.6.5_Linux_arm64.tar.gz) for Linux
3. Move it to a location in the PATH. For example, `mv  testkube_0.6.5_Linux_arm64/kubectl-testkube /usr/local/bin/kubectl-testkube`.

For Windows, you will need to unpack the binary and add it to `%PATH%` as well.

If you use a package manager that we don't support, please let us know here [#161](https://github.com/kubeshop/testkube/issues/161).


# **2. Testkube Server Componets**
To deploy Testkube to your K8s cluster you will need the following packages installed:
- [Kubectl docs](https://kubernetes.io/docs/tasks/tools/) 
- [Helm docs](https://helm.sh/docs/intro/install/#through-package-managers)


## Using Testkube's CLI to deploy the Server Components
The Testkube CLI provides an command to deploy the Testkube server components to your cluster easly.
Run:
```shell
testkube init
```
note: you must have your KUBECONFIG ponting to the desired location of the installation.

The above command will install the following components in your Kubernetes cluster:

1. Testkube API
2. `testkube` namespace
3. CRDs for Tests, TestSuites, Executors
4. MongoDB
5. Minio - default (can be disabled with `--no-minio` flag if you want to use S3 buckets)
6. Dashboard - default (can be disabled with `--no-dasboard` flag)


Confirm that Testkube is running:

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

By default Testkube is installed in the `testkube` namespace.

## **Using HELM to deploy the Server Components**
1. Add the Kubeshop Helm repository as follows:
```sh
helm repo add testkube https://kubeshop.github.io/helm-charts
```

If this repo already exists, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
testkube` to see the charts.

2. To install the `testkube` chart:

```sh
helm install --create-namespace my-testkube testkube/testkube
```

Please note that, by default, the namespace for the intstallation will be `testkube`. If the `testkube` namespace does not exist, it will be created for you.

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

### **Helm Properties**

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
| testkube-api.mongoDSN                | yes         | "mongodb://testkube-mongodb:27017"   |
| testkube-api.telemetryEnabled        | yes         | true                                 |
| testkube-api.storage.endpoint        | yes         | testkube-minio-service-testkube:9000 |
| testkube-api.storage.accessKeyId     | yes         | minio                                |
| testkube-api.storage.accessKey       | yes         | minio123                             |
| testkube-api.storage.scrapperEnabled | yes         | true                                 |
| testkube-api.slackToken              | yes         | ""                                   |
| testkube-api.slackChannelId          | yes         | ""                                   |

>For more configuration parameters of `MongoDB` chart please visit:
<https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters>

## **Remove Testkube Server Components**
Using helm
```sh
helm delete testkube
```
Using Testkube's CLI
```sh
testkube purge
```

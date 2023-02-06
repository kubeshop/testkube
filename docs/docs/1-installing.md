---
sidebar_position: 1
sidebar_label: Installation
---
# Installation Steps

<iframe width="100%" height="315" src="https://www.youtube.com/embed/ynzEkOUhxKk" title="YouTube Tutorial: Getting started with Testing in Kubernetes Using Testkube" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

In this section you will:

1. Install the Testkube CLI,
2. Use HELM or the Testkube CLI to install Testkube Server components in your cluster,
3. (Optional) Configure Testkube's Dashboard UI Ingress for your ingress-controller, if needed.

Watch the full installation video from our product experts: [Testkube Installation Video](https://www.youtube.com/watch?v=bjQboi3Etys).

## **1. Installing the Testkube CLI**

To install Testkube you'll need the following tools:

- [Kubectl](https://kubernetes.io/docs/tasks/tools/), Kubernetes command-line tool
- [Helm](https://helm.sh/)

Installing the Testkube CLI with Chocolatey and Homebrew will automatically install these dependencies if they are not present. For Linux-based systems please install them manually in advance.

### **MacOS**

```bash
brew install testkube
```

### **Windows**

```bash
choco source add --name=kubeshop_repo --source=https://chocolatey.kubeshop.io/chocolatey  
choco install testkube -y
```

### **Linux**

```bash
wget -qO - https://repo.testkube.io/key.pub | sudo apt-key add - && echo "deb https://repo.testkube.io/linux linux main" | sudo tee -a /etc/apt/sources.list && sudo apt-get update && sudo apt-get install -y testkube
```

### **Manual Download**

If you don't want to use scripts or package managers you can always do a manual install:

1. Download the binary for the version and platform of your choice [here](https://github.com/kubeshop/testkube/releases)
2. Unpack it. For example, in Linux use (tar -zxvf testkube_1.5.1_Linux_arm64.tar.gz)
3. Move it to a location in the PATH. For example, `mv  testkube_0.6.5_Linux_arm64/kubectl-testkube /usr/local/bin/kubectl-testkube`.

For Windows, you will need to unpack the binary and add it to the `%PATH%` as well.

If you use a package manager that we don't support, please let us know here [#161](https://github.com/kubeshop/testkube/issues/161).

## **2. Installing Testkube Server Components**

To deploy Testkube to your K8s cluster you will need the following packages installed:

- [Kubectl docs](https://kubernetes.io/docs/tasks/tools/)
- [Helm docs](https://helm.sh/docs/intro/install/#through-package-managers)

### **Using Testkube's CLI to Deploy the Server Components**

The Testkube CLI provides a command to easily deploy the Testkube server components to your cluster.
Run:

```bash
testkube init
```

note: you must have your KUBECONFIG pointing to the desired location of the installation.

The above command will install the following components in your Kubernetes cluster:

1. Testkube API
2. `testkube` namespace
3. CRDs for Tests, TestSuites, Executors
4. MongoDB
5. Minio - default (can be disabled with `--no-minio`)
6. Dashboard - default (can be disabled with `--no-dashboard` flag)

Confirm that Testkube is running:

```bash
kubectl get all -n testkube
```

By default Testkube is installed in the `testkube` namespace.

### **Using HELM to Deploy the Server Components**

1. Add the Kubeshop Helm repository as follows:

```bash
helm repo add testkube https://kubeshop.github.io/helm-charts
```

If this repo already exists, run `helm repo update` to retrieve
the `latest` versions of the packages.  You can then run `helm search repo
testkube` to see the charts.

2. To install the `testkube` chart:

```bash
helm install --create-namespace my-testkube testkube/testkube
```

Please note that, by default, the namespace for the installation will be `testkube`. If the `testkube` namespace does not exist, it will be created for you.

If you wish to install into a different namespace, please use following command:

```bash
helm install --namespace namespace_name my-testkube testkube/testkube
```

To uninstall the `testkube` chart if it was installed into default namespace:

```bash
helm delete my-testkube testkube/testkube
```

And from a namespace other than `testkube`:

```bash
helm delete --namespace namespace_name my-testkube testkube/testkube
```

#### **Helm Properties**

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

## **Remove Testkube Server Components**

### **Using Helm:**

```bash
helm delete testkube
```

### **Using Testkube's CLI:**

```bash
testkube purge
```

## Installation on OpenShift

Because of upgrade issues from Mongo 11 to 13, Testkube can't work on root-less OpenShift environment by default. Fortunately, you'll be able to install it manually.

To do it you need an empty OpenShift cluster and follow the steps below:

1. Save mongo chart values (named `values.yaml`)

```yaml
securityContext:
  enabled: true
  fsGroup: 1000650001
  runAsUser: 1000650001

podSecurityContext:
  enabled: false

containerSecurityContext:
  enabled: true
  runAsUser: 1000650001
  runAsNonRoot: true

volumePermissions:
  enabled: false

auth: 
   enabled: false
```

2. Install MongoDB

```sh
helm install testkube-mongodb bitnami/mongodb --namespace=testkube --values values.yaml
```

3. Install Testkube configured to use our Custom MongoDB instance

```
helm install --create-namespace --namespace testkube testkube testkube/testkube --set mongodb.enabled=false --set testkube-dashboard.service.port=8080
```

Please notice that since we've just installed MongoDB with a `testkube-mongodb` Helm release name, you are not required to reconfigure the Testkube API MongoDB connection URI. If you've installed with a different name/namespace, please adjust `--set testkube-api.mongodb.dsn: "mongodb://testkube-mongodb:27017"` to your MongoDB service.

## Installation with S3 Storage and IAM Authentication

To use S3 as storage, the steps are as follows:

1. Create a ServiceAccount with the ARN specified.
e.g.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::265500248336:role/minio-example
  name: s3-access
  namespace: testkube
```

2. In the Helm values.yaml file, link the ServiceAccount to the `testkube-api.minio.serviceAccountName` and to `testkube-api.jobServiceAccountName` then leave `minio.minioRootUser`, `minio.minioRootPassword` and `storage.port` empty and set `storage.endpoint` to `s3.amazonaws.com`.

3. Install using Helm and the values file with the above modifications.

## Configure Logging Storage
There are two types of storage mongo and minio, more details [here](13-Logging.md)

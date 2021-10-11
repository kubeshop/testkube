# Installation 

To get TestKube up and running you will need to

1. Install the kubectl testkube plugin
2. Install TestKube in your cluster 
3. Configure TestKube's Dashboard UI Ingress for your ingress-controller if needed.

## Install the kubectl testkube plugin

To install on Linux or MacOs run 
```sh
curl -sSLf https://kubeshop.github.io/testkube/install.sh | sudo bash
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

By default testkube is installed in `default` namespace but you can change it in manual install if you want.

If you want testkube to provide the endpoint for the kubest dashboard use `kubectl testkube install -i` with the `-i` or `--ingress` option, it will setup a ingress-nginx controller for you in a managed cluster(for baremetal clusters this should be set up manually before installing testkube).

## TestKube's Dashboard Ingress Configuration

Dashboard will bring you web-based UI for managing and seeing all the tests and its results via web-browser.
### Enabling dashboard
In order to enable dashboard please provide Helm's set value as follow during installation:
```
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true"
```
By default it's disabled
### configuration for the nginx-based ingress controller with the cert-manager pointed at Let'sencrypt. 
```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    acme.cert-manager.io/http01-edit-in-place: "true"
    cert-manager.io/cluster-issuer: letsencrypt-prod
    kubernetes.io/ingress.class: nginx
    kubernetes.io/ingress.global-static-ip-name: testkube-demo
    meta.helm.sh/release-name: testkube-demo
    meta.helm.sh/release-namespace: testkube-demo
    nginx.ingress.kubernetes.io/cors-allow-credentials: "false"
    nginx.ingress.kubernetes.io/cors-allow-methods: GET
    nginx.ingress.kubernetes.io/enable-cors: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "false"
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
  name: testkube-dashboard-testkube-demo
  namespace: testkube-demo
spec:
  rules:
  - host: your.domain.name
    http:
      paths:
      - backend:
          service:
            name: testkube-dashboard
            port:
              number: 80
        path: /
        pathType: Prefix
  tls:
  - hosts:
    - demo.testkube.io
    secretName: testkube-demo-cert-secret # to be created by your certificate manager within k8s cluster.
```
If you don't need TLS enabled just omit TLS configuration part. 

> Though we highly discourage working in non-safe environment whcih is exposed without usage of TLS-based connection. Please do so only in private internal environemnt for testing or development purposes only.

> Dashbaord talks to api-server via endpoint. Hence api-server will hvae to have DNS as well. 

Please note that you can install ingress for dashboard together with api-server ingress with the usage of Helm chart as well:
```
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true" --set testkube-dashboard.ingress.enabled="true" --set api-server.ingress.enabled="true"
```
If you need to specify some specific to your ingress annotations, you can use Helm "--set" option to pass needed annotations. E.G.:
```
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true" --set testkube-dashboard.ingress.enabled="true" --set api-server.ingress.enabled="true" --set api-server.ingress.annotations.kubernetes\\.io/ingress\\.class="anything_needed"
```
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
| api-server.postmanExecutorURI | yes | "http://testkube-postman-executor:8082" |
| postman-executor.image.repository | yes | "kubeshop/testkube-postman-executor" |
| postman-executor.image.pullPolicy | yes | "Always" |
| postman-executor.image.tag | yes | "latest" |
| postman-executor.service.type | yes | "NodePort" |
| postman-executor.service.port | yes | 8082 |
| postman-executor.mongoDSN | yes | "mongodb://testkube-mongodb:27017" |
| postman-executor.apiServerURI | yes | "http://testkube-api-server:8088" |

>For more configuration parameters of `MongoDB` chart please look here:
https://github.com/bitnami/charts/tree/master/bitnami/mongodb#parameters

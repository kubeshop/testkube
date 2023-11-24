# Guide to Deploying Testkube from Private Registries  

This guide shows how to deploy Testkube using images from private registries. 

To start with, we need to update `values.yaml` file, populating `registry` and `pullSecret` parameters with a value of your private registry and a k8s secret respectively. (Please note that the [k8s secret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/) should be created prior to Testkube installation.) 
The easiest solution would be to update `global` parameters, which will set a new value for **all** Testkube components, including MongoDB images:

```aidl
global:
  imageRegistry: ""
  imagePullSecrets: []
  labels: {}
  annotations: {}
```
However, NATS chart that is part of Testkube belongs to a third party and as of now it requires passing image registry and image pull secret parameters separately. The snippet from the `values.yaml` file for NATS chart:
```aidl
nats:
    imagePullSecrets: 
       - name: your-secret-name
    nats:
        image:
            registry: REGISTRY_NAME 
    natsbox:
        image:
            registry: REGISTRY_NAME  
    reloader:
        image:
            registry: REGISTRY_NAME  
    exporter:
        image:
            registry: REGISTRY_NAME
```

:::caution

Please mind that `global` parameters override all local values, so if it is required to set different registries or secret names, please use `registry` and `pullSecret` parameter for each Testkube service. For example `testkube-api`:
```aidl
testkube-api:
   image: 
     registry: your-registry-name
     repository: kubeshop/testkube-api-server
     tag: "latest"
     pullPolicy: IfNotPresent
     pullSecret: 
       - your-secret-name

```
:::

Once the `values.yaml` is ready we may deploy Testkube to the k8s cluster:
```aidl
helm repo add kubeshop https://kubeshop.github.io/helm-charts
helm install --create-namespace testkube kubeshop/testkube --namespace testkube --values ./path-to-values.yaml
```
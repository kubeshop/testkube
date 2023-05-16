# Ingress-NGINX

## Prerequisites

Add the repo to Helm:

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts && helm repo update
```

A values file for guidance can be found [here](https://github.com/kubeshop/helm-charts/blob/39f73098630b333ba66db137e7fc016c39d92876/testkube/charts/testkube/values-demo.yaml).

## Configure Ingress-NGINX to Expose Testkube API

The Testkube Dashboard needs the Testkube API to be exposed. For this, you can run:

```sh
helm upgrade testkube kubeshop/testkube --set testkube-api.ingress.enabled="true"
```

By default, the API's entry point is the path `/results`, so the results will be accessible at `$INGRESS_HOST/results/`

The Ingress configuration used is available in the [Testkube Helm Repo](https://github.com/kubeshop/helm-charts).

## Exposing the Testkube Dashboard 

To expose the Dashboard and the API together, run: 

```sh
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true" --set testkube-dashboard.ingress.enabled="true" --set testkube-api.ingress.enabled="true"
```

To get the address of Ingress use:

```sh
kubectl get ingress -n testkube
```

## HTTPS/TLS Configuration

To have secure access to Testkube Dashboard and the API, a certificate should be provided. The Helm charts can be configured from the Ingress section of the values file:

```yaml
ingress:
    enabled: "true"
    annotations: 
      kubernetes.io/ingress.class: nginx
      nginx.ingress.kubernetes.io/force-ssl-redirect: "false"
      nginx.ingress.kubernetes.io/ssl-redirect: "false"
      nginx.ingress.kubernetes.io/enable-cors: "true"
      nginx.ingress.kubernetes.io/cors-allow-methods: "GET"
      nginx.ingress.kubernetes.io/cors-allow-credentials: "false"
      # add an annotation indicating the issuer to use.
      cert-manager.io/cluster-issuer: letsencrypt-prod
      # controls whether the ingress is modified ‘in-place’,
      # or a new one is created specifically for the HTTP01 challenge.
      acme.cert-manager.io/http01-edit-in-place: "true"
    path: /
    hosts:
      - demo.testkube.io
    tlsenabled: "true"
    tls: # < placing a host in the TLS config will indicate a certificate should be created
    - hosts:
      - demo.testkube.io
      secretName: testkube-demo-cert-secret
```

Certificates are automatically generated using Let's Encrypt and cert-manager, but can be configured for any particular case. A full values file example can be found [here](https://github.com/kubeshop/helm-charts/blob/39f73098630b333ba66db137e7fc016c39d92876/testkube/charts/testkube/values-demo.yaml).

If there is no need for a TLS (Transport Layer Security) to be enabled, omit the TLS configuration.

:::important
We highly discourage working in a non-safe environment which is exposed without the use of a TLS-based connection. Please do so in a private internal environment for testing or development purposes only.
:::

To pass specific values to the Ingress annotations, the Helm "--set" option can be used: 

```sh
helm install testkube kubeshop/testkube --set testkube-dashboard.enabled="true" --set testkube-dashboard.ingress.enabled="true" --set testkube-api.ingress.enabled="true" --set testkube-api.ingress.annotations.kubernetes\\.io/ingress\\.class="anything_needed" 
```

A better approach is to configure and call a values file with the Ingress custom values:

```sh
helm install testkube kubeshop/testkube --values https://github.com/kubeshop/helm-charts/blob/39f73098630b333ba66db137e7fc016c39d92876/testkube/charts/testkube/values-demo.yaml
```

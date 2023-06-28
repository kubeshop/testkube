# Exposing Testkube Dashboard with NGINX Ingress

Usually, you would want to share the Testkube Dashboard with your internal company VPN to allow access to other team members without having to provide them access to your Kubernetes cluster. This is achieved by exposing the Testkube Dashboard. 

In this section we cover multiple solutions for different cloud providers.

## Prerequisites

1. Deploy NGINX Ingress Controller into your k8s cluster. Please see the link for more details: [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/).

2. (Optional) To use TLS, the installation of any certificate management tool is required. At Testkube and in this guide we will use [cert-manager](https://cert-manager.io/), but it might differ depending on your set-up.

3. Add the Testkube helm-chart to your repositories, using this command:

```sh
helm repo add kubeshop https://kubeshop.github.io/helm-charts && helm repo update
```

## Exposing Testkube

In order to expose Testkube to the outside world we need to enable two Ingresses - Testkube's UI API and Testkube's dashboard. Update the `values.yaml` file that will later be passed to the `helm install` command. To enable the Testkube Ingresses, please use the following code configuration:

```aidl
testkube-api:
  uiIngress:
    enabled: true
    annotations:
      kubernetes.io/ingress.class: nginx
      nginx.ingress.kubernetes.io/force-ssl-redirect: "false"
      nginx.ingress.kubernetes.io/ssl-redirect: "false"
      nginx.ingress.kubernetes.io/rewrite-target: /$1
      cert-manager.io/cluster-issuer: letsencrypt-prod
      acme.cert-manager.io/http01-edit-in-place: "true"
    path: /results/(v\d/.*)
    hosts:
      - your-host.com
    tlsenabled: "true"
    tls:
      - hosts:
          - your-host.com
        secretName: testkube-cert-secret


testkube-dashboard:
  ingress:
    enabled: "true"
    annotations:
      kubernetes.io/ingress.class: nginx
      nginx.ingress.kubernetes.io/force-ssl-redirect: "false"
      nginx.ingress.kubernetes.io/ssl-redirect: "false"
      cert-manager.io/cluster-issuer: letsencrypt-prod
      acme.cert-manager.io/http01-edit-in-place: "true"
    path: /
    hosts:
      - your-host.com
    tlsenabled: "true"
    tls:
      - hosts:
          - your-host.com
        secretName: testkube-cert-secret

  apiServerEndpoint: "your-host.com/results"
```

:::note

Keep in mind that hosts have to be identical for the `dashboard` and for the `api` with different paths.

Also, do not forget to add `apiServerEndpoint` to the `values.yaml` for the `testkube-dashboard`, e.g.: apiServerEndpoint: "your-host.com/results".

:::

Please note that the snippet includes annotations for cert manager as well. Certificates are automatically generated using Let's Encrypt and cert-manager, but can be configured for any particular case. If there is no need for a TLS (Transport Layer Security) to be enabled, omit the TLS configuration.

:::important

We highly discourage working in a non-safe environment which is exposed without the use of a TLS-based connection. Please do so in a private internal environment for testing or development purposes only.

:::

## Deployment

Once the `values.yaml` file is ready, we can deploy Testkube into a cluster:

```aidl
helm install --create-namespace testkube kubeshop/testkube --namespace testkube --values values.yaml
```

After the installation is complete, discover the address of the Ingresses with the following commands:

```sh
kubectl get ingress -n testkube
```

By default, the API's entry point is the path `/results`, so the results will be accessible at `$INGRESS_HOST/results/`

The Ingress configuration used is available in the [Testkube Helm Repo](https://github.com/kubeshop/helm-charts).

A values file for guidance can be found [here](https://github.com/kubeshop/helm-charts/blob/main/charts/testkube/values-demo.yaml#L334).

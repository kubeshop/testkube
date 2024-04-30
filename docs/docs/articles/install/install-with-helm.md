# Install Testkube with Helm

The Testkube Helm Chart lets you set up a customized Testkube for your environment.

## Prerequisites

Before you proceed with the installation, please ensure that you have the following prerequisites in place:

- Kubernetes cluster (version 1.21+)
- [Helm](https://helm.sh/docs/intro/quickstart/) (version 3+)
- [cert-manager](https://cert-manager.io/docs/installation/) (version 1.11+) - Used for TLS certificate management.
- [NGINX Controller](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/) (version v1.8+) - Used for Ingress configuration.
- (OPTIONAL) [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) (version 0.49+) - used for metrics collection
- Own a public/private domain for creating Ingress rules.
- License Key and/or License File, if offline access is required.

## Installing

1. Add the Testkube Helm repository:

```bash
helm repo add testkubeenterprise https://kubeshop.github.io/testkube-cloud-charts
```

2. Create and configure your values.yaml:

```bash
helm show values testkubeenterprise/testkube-enterprise > values.yaml

# Alternatively, export one of our profiles
testkube init <profile> --export > values.yaml
```

3. Install the Testkube Helm Chart:

```bash
helm upgrade --install \
  --create-namespace \
  --namespace testkube \
   -f values.yaml \
  testkube testkubeenterprise/testkube-enterprise
```

## General Settings

### License

You will have to set a license key to get started with Testkube. You can also opt to use a shared secret for your license. You can request a free license, no up-front credit card required.

```bash
global:
	enterpriseLicense: XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX-V3
```

### Domain

You should enable ingress and configure your domain to access Testkube services:

```
global:
  ingress:
    enabled: true
  domain: "example.com"
```

By default, the following services will be exposed. You can also choose to override the subdomain for each service.

| Service          | Default            | Override                       |
| ---------------- | ------------------ | ------------------------------ |
| Dashboard        | dashboard.$domain  | global.uiSubdomain             |
| REST API         | api.$domain        | global.restApiSubdomain        |
| gRPC API         | agent.$domain      | global.grpcApiSubdomain        |
| WebSockets API   | websockets.$domain | global.websocketsApiSubdomain  |
| Status Pages API | status.$domain     | global.statusPagesApiSubdomain |
| Storage API      | storage.$domain    | global.storageApiSubdomain     |

TLS can either be handled through cert-manager or a manually defined secret. While custom TLS certificates are possible, we strongly recommend using `cert-manager` for simplified certificate manager.

:::info
TLS should be terminated at the application-level instead of the ingress-level as it will drastically enhance the performance of the gRPC and Websocket protocols. Be mindful that NGINX defaults to a downgrade from HTTP2 to HTTP1.1 when the backend service is using an insecure port.
:::

#### Custom Certificates

In order to use custom certificates, first a secret needs to be created with the following entries:

- `tls.crt` - the certificate
- `tls.key` - the private key
- `ca.crt` - the CA certificate (if the certificate is not self-signed)

If certificate-based authentication is required, the custom certificates need to be configured in the following places:

- Enterprise API
  - If `MINIO_ENDPOINT` is set to an exposed URL, then the following Helm values need to be configured:
    - The following Helm parameter needs to be enabled to inject the custom certificate into MinIO `testkube-cloud-api.minio.certSecret.enabled: true`.
    - If the certificate is not self-signed, the CA cert needs to be injected also by enabling the Helm parameter `testkube-cloud-api.minio.mountCACertificate: true`.
    - Custom certificate verification can also be skipped by setting `testkube-cloud-api.minio.skipVerify: true`.
  - If `MINIO_ENDPOINT` uses the Kubernetes DNS record (`testkube-enterprise-minio.<namespace>.svc.cluster.local:9000`), `AGENT_STORAGE_HOSTNAME` should be set to point to the exposed storage URL.
- Agent
  - Agent API
    - If the Enterprise API is configured to use certificate-based authentication or is using a certificate signed by a custom CA, the Agent API needs to be configured to use the same certificates by pointing `testkube-api.cloud.tls.certificate.secretRef` to the Kubernetes secret which contains the certificates.
    - Custom certificate verification can also be skipped by setting `testkube-api.cloud.tls.skipVerify: true`.
  - Storage
    - The following Helm parameter needs to be enabled to inject the custom certificate into MinIO `testkube-api.storage.certSecret.enabled: true`.
    - If the certificate is not self-signed, the CA cert needs to be injected also by enabling the Helm parameter `testkube-cloud-api.minio.mountCACertificate: true`.
    - Custom certificate verification can also be skipped by setting `testkube-api.storage.skipVerify: true`.

### Auth

You will have to choose which users can access Testkube. Testkube uses Dex which providers the following identity providers:

- [Google](https://docs.testkube.io/testkube-pro-on-prem/articles/auth/#google)
- [Okta](https://docs.testkube.io/testkube-pro-on-prem/articles/auth/#azure-ad)
- [OIDC](https://docs.testkube.io/testkube-pro-on-prem/articles/auth/#okta)
- [Azure AD](https://docs.testkube.io/testkube-pro-on-prem/articles/auth/#azure-ad)
- [Other Dex connectors](https://dexidp.io/docs/connectors/), such as LDAP, BitBucket, OpenShift, etc…

Alternatively, you can use [a local database with static users](https://docs.testkube.io/testkube-pro-on-prem/articles/auth/#static-users) which acts as a virtual identity provider for initial evaluations.

Once authenticated, users will also need to be invited to org. By default, new users will automatically join the default organization. You can change this behaviour by changing the bootstrap and invitation configuration.

### Telemetry

Testkube exposes Prometheus metrics on the `/metrics` endpoint and uses a `ServiceMonitor` resource to expose them to Prometheus. In order for this to work, you need to have `Prometheus Operator` installed in your cluster so that the `ServiceMonitor` resource can be created.

Use the following configuration to enable metrics:

```helm
testkube-cloud-api:
  prometheus:
    enabled: true
```

## Shared Secrets

Testkube requires a variety of secrets to operate. Any required secret that is not provided manually will be automatically generated. You can optionally choose to specify your own secrets. For the current shared secrets, it’s recommended to use the autogeneration.

- testkube-default-agent-token
- testkube-minio-credentials
- testkube-license

[license]: https://testkube.io/download

## Advanced Settings

Check out [this article][advanced] to learn more about our advanced settings. You can learn about organization management, custom ingress controllers, production environments and more.

[advanced]: /articles/install/advanced-install

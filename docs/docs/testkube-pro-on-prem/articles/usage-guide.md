# Helm Chart Installation and Usage Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
- [Testkube Pro On-Prem Helm Chart Installation and Usage Guide](#testkube-enterprise-helm-chart-installation-and-usage-guide)
  - [Installation of Testkube Enterprise and an Agent in the same cluster](#installation-of-testkube-enterprise-and-an-agent-in-the-same-cluster)
  - [Installation of Testkube Enterprise and an Agent in multiple clusters](#installation-of-testkube-enterprise-and-an-agent-in-multiple-clusters)
  - [Prerequisites](#prerequisites)
  - [Configuration](#configuration)
    - [Docker images](#docker-images)
    - [License](#license)
      - [Online License](#online-license)
      - [Offline License](#offline-license)
    - [Ingress](#ingress)
      - [Configuration](#configuration-1)
      - [Domain](#domain)
      - [TLS](#tls)
    - [Auth](#auth)
    - [Metrics](#metrics)
    - [Invitations](#invitations)
      - [Invitations via email](#invitations-via-email)
      - [Auto-accept invitations](#auto-accept-invitations)
    - [Organization and Environment Management](#organization-and-environment-management)
  - [Bring Your Own Infra](#bring-your-own-infra)
    - [MongoDB](#mongodb)
    - [NATS](#nats)
    - [MinIO](#minio)
    - [Dex](#dex)
  - [Installation](#installation)
    - [Production setup](#production-setup)
  - [FAQ](#faq)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


Welcome to the Testkube Enterprise Helm chart installation and usage guide.
This comprehensive guide provides step-by-step instructions for installing and utilizing the Testkube Enterprise Helm chart.
Testkube Enterprise is a cutting-edge Kubernetes-native testing platform designed to optimize your testing and quality assurance processes with enterprise-grade features.

## Installation of Testkube Enterprise and an Agent in the same cluster

We have a simplified installation process to allow to deploy everything in a single cluster. You can find all the details at [the Testkube Quickstart](../../articles/install/quickstart-install.mdx).


## Installation of Testkube Enterprise and an Agent in multiple clusters
## Prerequisites

Before you proceed with the installation, please ensure that you have the following prerequisites in place:
* Kubernetes cluster (version 1.21+)
* [Helm](https://helm.sh/docs/intro/quickstart/) (version 3+)
* [cert-manager](https://cert-manager.io/docs/installation/) (version 1.11+) - Used for TLS certificate management.
* [NGINX Controller](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/) (version v1.8+) - Used for Ingress configuration.
* (OPTIONAL) [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) (version 0.49+) - used for metrics collection
* Own a public/private domain for creating Ingress rules.
* License Key and/or License File, if offline access is required.

**NOTE**
While it is possible to use custom TLS certificates for the Testkube Enterprise API and Dashboard,
we strongly recommend using `cert-manager` for easier certificate management.

## Configuration

### Docker Images

**DEPRECATION NOTICE**: As of November 2023, Testkube Enterprise Docker images are publicly accessible.
You only need to follow the steps in this section if you wish to re-publish the images to your private Docker registry;
otherwise, you may skip this section.

To begin, ensure that you have access to the Testkube Enterprise API & Dashboard Docker images.
You can either request access from your Testkube representative or upload the Docker image tarball artifacts to a private Docker registry.

Next, create a secret to store your Docker registry credentials:
```bash
kubectl create secret docker-registry testkube-enterprise-registry \
  --docker-server=<your-registry-server> \
  --docker-username=<your-name>          \
  --docker-password=<your-pword>         \
  --docker-email=<your-email>            \
  --namespace=testkube-enterprise
```

Make sure to configure the image pull secrets in your `values.yaml` file:
```helm
global:
  imagePullSecrets:
    - name: testkube-enterprise-registry
```

### License

Select the appropriate license type for your environment.

For air-gapped & firewalled environments, we offer an option to use an [Offline License](#offline-license) for enhanced security.
An **Offline License** consists of a **License Key** and **License File**.

If your environment has internet access, you can use an [Online License](#online-license), which only requires the **License Key**.

#### Online License

If your environment has internet access, you can use an **Online License**, which only requires the **License Key**,
and can be provided as a Helm parameter or Kubernetes secret.

To provide the **License Key** as a Helm parameter, use the following configuration:
```helm
global:
  enterpriseLicenseKey: <your license key>
```

To provide the **License Key** as a Kubernetes secret, first we need to create a secret with the required field.
Run the following command to create the secret:
```bash
kubectl create secret generic testkube-enterprise-license \
  --from-literal=LICENSE_KEY=<your license key>           \
  --namespace=testkube-enterprise
```
And then use the following Helm chart configuration:
```helm
global:
  enterpriseLicenseSecretRef: <secret name>
```

#### Offline License

For an **Offline License**, supply both the **License Key** and **License File** as either Kubernetes secrets or Helm parameters.
Using secrets is safer, as it prevents exposing sensitive license information in Helm chart values.

The Kubernetes secret needs to contain 2 entries: `license.lic` and `LICENSE_KEY`.
To create the secret with the **License Key** and **License File**, run the following command:
```bash
kubectl create secret generic testkube-enterprise-license \
  --from-literal=LICENSE_KEY=<your license key>            \
  --from-file=license.lic=<path-to-license-file>          \
  --namespace=testkube-enterprise
```

After creating the secret, use the following Helm chart configuration:
```helm
global:
  enterpriseOfflineAccess: true
  licenseFileSecret: testkube-enterprise-license
```

Alternatively, you can provide the **License File** as a Helm parameter:
```helm
global:
  licenseKey: <your license key>
  licenseFile: <your license file>
```

### Ingress

Testkube Enterprise requires the NGINX Controller to configure and optimize its protocols.
NGINX is the sole supported Ingress Controller, and is essential for Testkube Enterprise's operation.


We highly recommend installing Testkube Enterprise with Ingress enabled.
This requires a valid domain (public or private) with a valid TLS certificate.
Ingresses are enabled and created by default.

To disable Ingress creation, adjust the following values accordingly. Note that you must then manually configure the API & Dashboard services to maintain accessibility:
```helm
global:
  ingress:
    enabled: false

testkube-cloud-api:
  api:
    tls:
      serveHTTPS: false
```

#### Configuration

To ensure the reliable functioning of gRPC and Websockets protocols, Testkube Enterprise is locked in with NGINX Ingress Controller.

Below are current configurations per Ingress resource which ensure Testkube Enterprise protocols work performant and reliably.
It is not recommended to change any of these settings!

gRPC Ingress annotations:
```kubernetes
annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: 8m
    nginx.ingress.kubernetes.io/client-header-timeout: "10800"
    nginx.ingress.kubernetes.io/client-body-timeout: "10800"
```

Websockets Ingress annotations:
```kubernetes
annotations:
  nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
  nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
```

If you want to use a different Ingress Controller, please reach out to our support team.

#### Domain

Testkube Enterprise requires a domain (public or internal) under which it will expose the following services:

| Subdomain                       | Service          |
|---------------------------------|------------------|
| `dashboard.<your-(sub)domain>`  | Dashboard UI     |
| `api.<your-(sub)domain>`        | REST API         |
| `agent.(sub)<your-domain>`      | gRPC API         |
| `websockets.(sub)<your-domain>` | WebSockets API   |
| `storage.(sub)<your-domain>`    | Storage API      |
| `status.(sub)<your-domain>`     | Status Pages API |

#### TLS

For best the performance, TLS should be terminated at the application level (Testkube Enterprise API) instead of NGINX/Ingress level because
gRPC and Websockets protocols perform significantly better when HTTP2 protocol is used end-to-end.
Note that NGINX, by default, downgrades the HTTP2 protocol to HTTP1.1 when the backend service is using an insecure port.

If `cert-manager` (check the [Prerequisites](#prerequisites) for installation guide) is installed in your cluster, it should be configured to issue certificates for the configured domain by using the `Issuer` or `ClusterIssuer` resource.
Testkube Enterprise Helm chart needs the following config in that case:
```helm
global:
  certificateProvider: "cert-manager"
  certManager:
    issuerRef: <issuer|clusterissuer name>
```

By default, Testkube Enterprise uses a `ClusterIssuer` `cert-manager` resource, that can be changed by setting the `testkube-cloud-api.api.tls.certManager.issuerKind` field to `Issuer`.

If `cert-manager` is not installed in your cluster, valid TLS certificates (for API & Dashboard) which cover the following subdomains need to be provided:
* API (tls secret name is configured with `testkube-cloud-api.api.tls.tlsSecret` field)
    * `api.<your-domain>`
    * `agent.<your-domain>`
    * `websockets.<your-domain>`
    * `status.<your-domain>`
* Dashboard (TLS secret name is configured with `testkube-cloud-ui.ingress.tlsSecretName` field)
    * `dashboard.<your-domain>`
      Also, `global.certificateProvider` should be set to blank ("").
```helm
global:
  certificateProvider: ""
```

#### Custom certificates

In order to use custom certificates, first a secret needs to be created with the following entries:
* `tls.crt` - the certificate
* `tls.key` - the private key
* `ca.crt` - the CA certificate (if the certificate is not self-signed)

If certificate-based authentication is required, the custom certificates need to be configured in the following places:
* Enterprise API
  * If `MINIO_ENDPOINT` is set to an exposed URL, then the following Helm values need to be configured:
    - The following Helm parameter needs to be enabled to inject the custom certificate into MinIO `testkube-cloud-api.minio.certSecret.enabled: true`
    - If the certificate is not self-signed, the CA cert needs to be injected also by enabling the Helm parameter `testkube-cloud-api.minio.mountCACertificate: true`
    - Custom certificate verification can also be skipped by setting `testkube-cloud-api.minio.skipVerify: true`
  * If `MINIO_ENDPOINT` uses the Kubernetes DNS record (`testkube-enterprise-minio.<namespace>.svc.cluster.local:9000`), `AGENT_STORAGE_HOSTNAME` should be set to point to the exposed storage URL
* Agent
  * Agent API
    - If the Enterprise API is configured to use certificate-based authentication or is using a certificate signed by a custom CA, the Agent API needs to be configured to use the same certificates by pointing `testkube-api.cloud.tls.certificate.secretRef` to the Kubernetes secret which contains the certificates
    - Custom certificate verification can also be skipped by setting `testkube-api.cloud.tls.skipVerify: true`
  * Storage
    - The following Helm parameter needs to be enabled to inject the custom certificate into MinIO `testkube-api.storage.certSecret.enabled: true`
    - If the certificate is not self-signed, the CA cert needs to be injected also by enabling the Helm parameter `testkube-cloud-api.minio.mountCACertificate: true`
    - Custom certificate verification can also be skipped by setting `testkube-api.storage.skipVerify: true`

### Auth

Testkube Enterprise utilizes [Dex](https://dexidp.io/) for authentication and authorization.
For detailed instruction on configuring Dex, please refer to the [Identity Provider](./auth.md) document.

### Metrics

Testkube Enterprise exposes Prometheus metrics on the `/metrics` endpoint and uses a `ServiceMonitor` resource to expose them to Prometheus.
In order for this to work, you need to have `Prometheus Operator` installed in your cluster so that the `ServiceMonitor` resource can be created.


Use the following configuration to enable metrics:
```helm
testkube-cloud-api:
  prometheus:
    enabled: true
```

### Invitations

Testkube Enterprise allows you to invite users to Organizations and Environments within Testkube, granting them specific roles and permissions.

There are two supported invitation modes: `email` and `auto-accept`.
Use `email` to send an invitation for the user to accept, and `auto-accept` to automatically add users without requiring acceptance.

#### Invitations Via Email

If `testkube-cloud-api.api.inviteMode` is set to `email`, Testkube Enterprise will send emails when a user gets invited to
an Organization or an Environment and when SMTP settings need to be configured in the API Helm chart.

```helm
testkube-cloud-api:
  api:
    email:
      fromEmail: "example@gmail.com"
      fromName: "Example Invitation"
    inviteMode: email
    smtp:
      host: <smtp host>
      port: <smtp port>
      username: <smtp username>
      password: <smtp password>
      # password can also be referenced by using the `passwordSecretRef` field which needs to contain the key SMTP_PASSWORD
      # passwordSecretRef: <secret name>
```

#### Auto-accept Invitations

If `testkube-cloud-api.api.inviteMode` is set to `auto-accept`, Testkube Enterprise will automatically add users to
Organizations and Environments when they get invited.

```helm
testkube-cloud-api:
  inviteMode: auto-accept
```

### Organization and Environment Management

Testkube Pro On-Prem allows you to manage organizations and environments using configuration.

```helm
testkube-cloud-api:
  api:
    features:
      bootstrapConfig:
        enabled: true
        config:
          organizations:
            - name: prod_organization
              environments:
                - name: production_1
                - name: production_2
```

On startup, the `prod_organization` organization with two environments, `production_1` and `production_2` will be created.

Next, you can enhance the configuration to automatically add new users to organizations and environments with predefined roles. For example, the following config makes new users join `prod_organization` as a member role and use `production_1` environment as a run role:

```helm
      bootstrapConfig:
        enabled: true
        config:
          default_organizations:
            - prod_organization
          organizations:
            - name: prod_organization
              default_role: member
              default_environments:
                - production_1
              environments:
                - name: production_1
                  default_role: run
                - name: production_2
```
Note: The default organization and environment mapping only apply on first sign in. After, you can remove users from environments or change roles thru Testkube UI.

Additionally, by default, Testkube Pro creates a personal organization for every new user. When using default organization and environment configuration, you can turn off personal organizations using the following config:

```helm
testkube-cloud-api:
  api:
    features:
      disablePersonalOrgs: true
```

## Bring Your Own Infra

Testkube Enterprise supports integrating with existing infrastructure components such as MongoDB, NATS, Dex, etc.

### MongoDB

Testkube Enterprise uses MongoDB as a database for storing all the data.
By default, it will install a MongoDB instance using the Bitnami MongoDB Helm chart.

If you wish to use an existing MongoDB instance, you can configure the following values:
```helm
mongodb:
  enabled: false
 
testkube-cloud-api:
  api:
    mongo:
      dsn: <mongodb dsn (mongodb://...)>
```

### NATS

Testkube Enterprise uses NATS as a message broker for communication between API and Agents.

If you wish to use an existing NATS instance, you can configure the following values:
```helm
nats:
  enabled: false
  
testkube-cloud-api:
  api:
    nats:
      uri: <nats uri (nats://...)>
```

### MinIO

Testkube Enterprise uses MinIO as a storage backend for storing artifacts.

If you wish to use an existing MinIO instance, you can configure the following values:
```helm
testkube-cloud-api:
  minio:
    enabled: false
  api:
    minio: {} # check out the `testkube-cloud-api.api.minio` block in the values.yaml for all available settings
```

### Dex

Testkube Enterprise uses Dex as an identity provider.

If you wish to use an existing Dex instance, you can configure the following values:
```helm
global:
  dex:
    issuer: <dex issuer url>
dex:
  enabled: false
testkube-cloud-api:
  api:
    oauth: {} # check out the `testkube-cloud-api.api.oauth` block in the values.yaml for all available settings
```

## Installation

1. Add our Testkube Enterprise Helm registry:
    ```bash
    helm repo add testkubeenterprise https://kubeshop.github.io/testkube-cloud-charts
    ```
2. Create a `values.yaml` with preferred configuration.
3. Run `helm install testkube-enterprise testkubeenterprise/testkube-enterprise -f values.yaml --namespace testkube-enterprise`.

**IMPORTANT**
The Bitnami MongoDB Helm chart does not work reliably on ARM architectures. If you are installing MongoDB using this chart, you need to use an ARM compatible image:
```helm
mongodb:
  image:
    repository: zcube/bitnami-compat-mongodb
    tag: "6.0.5"
```


### Production Setup

For best performance and reliability, users should follow this official setup guide and make sure each section is properly configured.

1. Configure DNS records as described in the [Domain](#domain) section
2. Configure TLS certificates as described in the [TLS](#tls) section
3. Configure Dex as described in the [Auth](#auth) section
4. Configure Ingress as described in the [Ingress](#ingress) section
5. Configure Metrics as described in the [Metrics](#metrics) section
6. Configure Invitations as described in the [Invitations](#invitations) section
7. Configure BYOI components as described in the [Bring Your Own Infra](#bring-your-own-infra) section
8. Install Testkube Enterprise as described in the [Installation](#installation) section

## FAQ

Q: Testkube Enterprise API is crashing (pod is in `Error`/`CrashLoopBackOff` state) with the following error:
```
panic: license file is invalid
```
A: Make sure the license file ends with a newline character.
There should be a new line after the `-----END LICENSE FILE-----` line in the license file.

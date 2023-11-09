# Helm Chart Installation and Usage Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
- [Testkube Enterprise Helm Chart Installation and Usage Guide](#testkube-enterprise-helm-chart-installation-and-usage-guide)
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
    - [Invitations](#invitations)
        - [Invitations via email](#invitations-via-email)
        - [Auto-accept invitations](#auto-accept-invitations)
    - [Bring Your Own Infra](#bring-your-own-infra)
        - [MongoDB](#mongodb)
        - [NATS](#nats)
        - [MinIO](#minio)
        - [Dex](#dex)
    - [Installation](#installation)
        - [Minimal setup](#minimal-setup)
        - [Production setup](#production-setup)
    - [FAQ](#faq)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


Welcome to the Testkube Enterprise Helm chart installation and usage guide.
This comprehensive guide provides step-by-step instructions for installing and utilizing the Testkube Enterprise Helm chart.
Testkube Enterprise is a cutting-edge Kubernetes-native testing platform designed to optimize your testing and quality assurance processes with enterprise-grade features.

## Prerequisites

Before you proceed with the installation, please ensure that you have the following prerequisites in place:
* Kubernetes cluster (version 1.21+)
* [Helm](https://helm.sh/docs/intro/quickstart/) (version 3+)
* [cert-manager](https://cert-manager.io/docs/installation/) (version 1.11+) - used for TLS certificate management
* [NGINX Controller](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/) (version v1.8+) - used for Ingress configuration
* Own a public/private domain for creating Ingress rules
* License Key and/or License File (if offline access is required)

**NOTE**
While it is possible to use custom TLS certificates for the Testkube Enterprise API and Dashboard,
we strongly recommend using `cert-manager` for easier certificate management.

## Configuration

### Docker images

**NOTE**: As of November 2023, Testkube Enterprise Docker images are publicly accessible.
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

For air-gaped & firewalled environments, we offer an option to use an [Offline License](#offline-license) for enhanced security.
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
  nginx.ingress.kubernetes.io/server-snippets: |
    location / {
      proxy_set_header Upgrade $http_upgrade;
      proxy_http_version 1.1;
      proxy_set_header X-Forwarded-Host $http_host;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-For $remote_addr;
      proxy_set_header Host $host;
      proxy_set_header Connection "upgrade";
      proxy_cache_bypass $http_upgrade;
    }
```

If you want to use a different Ingress Controller, we kindly ask you to reach out and discuss with our support team.

#### Domain

Testkube Enterprise requires a domain (public or internal) under which it will expose the following services:
* Dashboard -> `https://dashboard.<your-domain>`
* REST API -> `https://api.<your-domain>`
* Websocket API -> `wss://websockets.<your-domain>`
* gRPC API -> `grpc://agent.<your-domain>`

#### TLS

For best performance, TLS should be terminated at application level (Testkube Enterprise API) instead of NGINX/Ingress level because
gRPC and Websockets protocols perform significantly better when HTTP2 protocol is used end-to-end.
Note that NGINX, by default, downgrades the HTTP2 protocol to HTTP1.1 when the backend service is using an insecure port.

If `cert-manager` (check the [Prerequisites](#prerequisites) for installation guide) is installed in your cluster, it should be configured to issue certificates for the configured domain by using `Issuer` or `ClusterIssuer` resource.
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
* Dashboard (TLS secret name is configured with `testkube-cloud-ui.ingress.tlsSecretName` field)
    * `dashboard.<your-domain>`
      Also, `global.certificateProvider` should be set to blank ("").
```helm
global:
  certificateProvider: ""
```

### Auth

Testkube Enterprise utilizes [Dex](https://dexidp.io/) for authentication & authorization.
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

## Invitations

Testkube Enterprise allows you to invite users to Organizations and Environments within Testkube, granting them specific roles and permissions.

There are two supported invitation modes: `email` and `auto-accept`.
Use `email` to send an invitation for the user to accept, and `auto-accept` to automatically add users without requiring acceptance.

### Invitations via email

If `testkube-cloud-api.api.inviteMode` is set to `email`, Testkube Enterprise will send emails when a user gets invited to
an Organization or an Environment, and in that case SMTP settings need to be configured in the API Helm chart.

```helm
testkube-cloud-api:
  api:
    inviteMode: email
    smtp:
      host: <smtp host>
      port: <smtp port>
      username: <smtp username>
      password: <smtp password>
      # password can also be referenced by using the `passwordSecretRef` field which needs to contain the key SMTP_PASSWORD
      # passwordSecretRef: <secret name>
```

### Auto-accept invitations

If `testkube-cloud-api.api.inviteMode` is set to `auto-accept`, Testkube Enterprise will automatically add users to
Organizations and Environments when they get invited.

```helm
testkube-cloud-api:
  inviteMode: auto-accept
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
2. Create a `values.yaml` with preferred configuration
3. Run `helm install testkube-enterprise testkubeenterprise/testkube-enterprise -f values.yaml --namespace testkube-enterprise`

**IMPORTANT**
The Bitnami MongoDB Helm chart does not work reliably on ARM architectures. If you are installing MongoDB using this chart, you need to use an ARM compatible image:
```helm
mongodb:
  image:
    repository: zcube/bitnami-compat-mongodb
    tag: "6.0.5"
```

### Minimal setup

This is a minimal setup which will install a development Testkube Enterprise cluster with the following components:
* Testkube Enterprise API
* Testkube Enterprise Dashboard
* MongoDB
* NATS
* Dex

This setup should not be used in production environments. For a more advanced setup please refer to the [Production Setup](#production-setup) section.

Following configuration can be used for a minimal development setup of Testkube Enterprise:
```helm
global:
  domain: <your domain>
  imagePullSecrets:
    - name: <docker creds secret>
  licenseKey: <your license key>
  ingress:
    enabled: false

dex:
  configTemplate:
    additionalConfig: |
      enablePasswordDB: true
      staticPasswords:
        - email: <user email>
          hash: <bcrypt hash of user password>
          username: <username>
```

### Production setup

TBD

## FAQ

Q: Testkube Enterprise API is crashing (pod is in `Error`/`CrashLoopBackOff` state) with the following error:
```
panic: license file is invalid
```
A: Make sure the license file ends with a newline character.
There should be a new line after the `-----END LICENSE FILE-----` line in the license file.
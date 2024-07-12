# Helm Chart Installation and Usage Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Components](#components)
- [Installation modes](#installation-modes)
  - [Demo single-cluster installation](#demo-single-cluster-installation)
  - [Multi-cluster installation](#multi-cluster-installation)
    - [Prerequisites](#prerequisites)
- [Configuration](#configuration)
  - [License](#license)
    - [Online License](#online-license)
    - [Offline License](#offline-license)
  - [Ingress](#ingress)
    - [Configuration](#configuration-1)
    - [Domain](#domain)
    - [TLS](#tls)
    - [Self-signed certificates](#self-signed-certificates)
  - [Auth](#auth)
  - [Metrics](#metrics)
  - [Invitations](#invitations)
    - [Invitations Via Email](#invitations-via-email)
    - [Auto-accept Invitations](#auto-accept-invitations)
  - [Organization and Environment Management](#organization-and-environment-management)
- [Bring Your Own Infra](#bring-your-own-infra)
  - [MongoDB](#mongodb)
  - [NATS](#nats)
  - [MinIO](#minio)
  - [Dex](#dex)
- [Installation](#installation)
  - [Production Setup](#production-setup)
- [FAQ](#faq)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

Welcome to the Testkube On-Prem Helm chart installation and usage guide.
This comprehensive guide provides step-by-step instructions for installing and utilizing the Testkube Enterprise Helm chart.
Testkube On-Prem is a cutting-edge Kubernetes-native testing platform designed to optimize your testing and quality assurance processes with enterprise-grade features.

## Components

Testkube On-Prem consists of the following components:
* Testkube Control Plane - The central component that manages connected Agents.
  * API - A service which runs the REST, Agent gRPC and Websocket APIs for interacting with the Control Plane.
    * Helm chart - Bundled as a subchart in the [testkube-enterprise](https://github.com/kubeshop/testkube-cloud-charts/tree/main/charts/testkube-enterprise) Helm chart.
    * Docker image - [kubeshop/testkube-enterprise-api](https://hub.docker.com/r/kubeshop/testkube-enterprise-api)
  * Dashboard - The web-based UI for managing tests, environments, and users.
    * Helm chart - Bundled as a subchart in the [testkube-enterprise](https://github.com/kubeshop/testkube-cloud-charts/tree/main/charts/testkube-enterprise) Helm chart. 
    * Docker image - [kubeshop/testkube-enterprise-ui](https://hub.docker.com/r/kubeshop/testkube-enterprise-ui)
  * Worker Service - A service which handles async operations for artifacts and test executions.
    * Helm chart - Bundled as a subchart in the [testkube-enterprise](https://github.com/kubeshop/testkube-cloud-charts/tree/main/charts/testkube-enterprise) Helm chart. 
    * Docker image - [kubeshop/testkube-enterprise-worker-service](https://hub.docker.com/r/kubeshop/testkube-enterprise-worker-service)
* Testkube Agent - A lightweight component that connects to the Control Plane and executes test runs.
  * Helm chart - [kubeshop/testkube](https://github.com/kubeshop/helm-charts/tree/main/charts/testkube)
  * Docker image - [kubeshop/testkube-api-server](https://hub.docker.com/r/kubeshop/testkube-api-server)

The Control Plane Helm charts are published in the [testkubeenterprise](https://kubeshop.github.io/testkube-cloud-charts) Helm registry.
Run the following command to add the Testkube On-Prem Helm registry:
```bash
helm repo add testkubeenterprise https://kubeshop.github.io/testkube-cloud-charts
```

The Agent Helm chart is published in the [kubeshop](https://kubeshop.github.io/helm-charts) Helm registry.
Run the following command to add the Testkube Helm registry:
```bash
helm repo add kubeshop https://kubeshop.github.io/helm-charts
```

For external dependencies, check the [Bring Your Own Infra](#bring-your-own-infra) section.

## Installation modes

Testkube On-Prem supports two installation modes: a demo single-cluster and a multi-cluster installation.

For a quick start, we recommend beginning with the demo single-cluster installation.
This setup installs both the Testkube Control Plane and an Agent within the same cluster.

Once you're familiar with this configuration, you can proceed to the multi-cluster installation,
which deploys the Testkube Control Plane in one cluster, while Agents can be installed in the same or different clusters.

### Demo single-cluster installation

We offer a demo installer which deploys Testkube On-Prem and connects an Agent in a single cluster.
You can find all the details at [the Testkube Quickstart](../../articles/install/quickstart-install.mdx).

### Multi-cluster installation

Multi-cluster installation is a more advanced setup that requires additional configuration.
This setup is recommended for production environments where you want to separate the Control Plane and Agents for better scalability and security.

This setup configures the Testkube Control Plane in a central cluster and exposes the necessary components so Agents can connect from the same or other clusters.

#### Prerequisites

Before you proceed with the installation, please ensure that you have the following prerequisites in place:
* Kubernetes cluster (version 1.21+)
* [Helm](https://helm.sh/docs/intro/quickstart/) (version 3+)
* (RECOMMENDED) [cert-manager](https://cert-manager.io/docs/installation/) (version 1.11+) - Used for TLS certificate management.
* (RECOMMENDED) [NGINX Controller](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/) (version v1.8+) - Used for Ingress configuration.
* (OPTIONAL) [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) (version 0.49+) - used for metrics collection
* Own a public/private domain for creating Ingress rules.
* License Key and/or License File, if offline access is required.

If your Kubernetes cluster is using Istio, please refer to the [Istio](../../articles/istio.md) article for additional configuration.

## Configuration

### License

Select the appropriate license type for your environment.

If your environment can access the Testkube On-Prem License Server (https://api.keygen.sh), you can use an **Online License** which only requires a **License Key**.

For air-gapped & firewalled environments, we offer an option to use an [Offline License](#offline-license) for enhanced security.
An **Offline License** consists of a **License Key** and **License File**.

#### Online License

To provide the **License Key** as a Helm parameter, use the following configuration:
```helm
global:
  enterpriseLicenseKey: <license key>
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
  enterpriseLicenseSecretRef: testkube-enterprise-license
```

#### Offline License

For an **Offline License**, supply both the **License Key** and **License File** as either Kubernetes secrets or Helm parameters.
Using secrets is safer, as it prevents exposing sensitive license information in Helm chart values.

The Kubernetes secret needs to contain 2 entries: `license.lic` and `LICENSE_KEY`.
To create the secret with the **License Key** and **License File**, run the following command:
```bash
kubectl create secret generic testkube-enterprise-license \
  --from-literal=LICENSE_KEY=<your license key>           \
  --from-file=license.lic=<path-to-license-file>          \
  --namespace=testkube-enterprise
```

After creating the secret, use the following Helm chart configuration:
```helm
global:
  enterpriseOfflineAccess: true
  enterpriseLicenseSecretRef: testkube-enterprise-license
```

### Ingress

Testkube On-Prem officially only supports the NGINX Controller as the default configuration relies on NGINX.
Other Ingress Controllers can also be used (i.e. Istio Ingress Controller), but they may require additional configuration.

We highly recommend installing Testkube On-Prem with Ingress enabled.
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

Below are current configurations per Ingress resource which ensure Testkube On-Prem protocols work performant and reliably.

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

Testkube On-Prem requires a domain (public or internal) under which it will expose the following services:

| Subdomain                       | Service          |
|---------------------------------|------------------|
| `dashboard.<your-(sub)domain>`  | Dashboard UI     |
| `api.<your-(sub)domain>`        | REST API         |
| `agent.(sub)<your-domain>`      | gRPC API         |
| `websockets.(sub)<your-domain>` | WebSockets API   |
| `storage.(sub)<your-domain>`    | Storage API      |

#### TLS

For best performance, it is recommended to terminate TLS at the application level (Testkube Control Plane) instead of NGINX/Ingress level because
gRPC and Websockets protocols perform significantly better when HTTP2 protocol is used end-to-end.
Note that NGINX, by default, downgrades the HTTP2 protocol to HTTP1.1 when the backend service is using an insecure port.

If `cert-manager` (check the [Prerequisites](#prerequisites) for installation guide) is installed in your cluster, it should be configured to issue certificates for the configured domain by using the `Issuer` or `ClusterIssuer` resource.
Testkube On-Prem Helm chart needs the following config in that case:
```helm
global:
  certificateProvider: "cert-manager"
  certManager:
    issuerRef: <issuer|clusterissuer name>
```

By default, Testkube On-Prem uses a `ClusterIssuer` `cert-manager` resource, that can be changed by setting the `testkube-cloud-api.api.tls.certManager.issuerKind` field to `Issuer`.

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

#### Self-signed certificates

If the Testkube On-Prem Control Plane components are behind a Load Balancer utilizing self-signed certificates, additional configuration must be provided to the Agent Helm chart during installation.
Use one of the following methods to configure the Agent Helm chart to trust the self-signed certificates:
1. Inject the custom CA certificate
    ```helm
    # testkube chart
    global:
      tls:
        caCertPath: /etc/testkube/certs
      volumes:
        additionalVolumes:
          - name: custom-ca
            secret:
              secretName: custom-cert
        additionalVolumeMounts:
          - name: custom-ca
            mountPath: /etc/testkube/certs
            readOnly: true
    ```
2. Skip TLS verification (not recommended in a production setup)
    ```helm
    # testkube chart
    global:
      tls:
        skipVerify: true
    ```


### Auth

Testkube On-Prem utilizes [Dex](https://dexidp.io/) for authentication and authorization.
For detailed instruction on configuring Dex, please refer to the [Identity Provider](./auth.md) document.

### Metrics

Testkube On-Prem exposes Prometheus metrics on the `/metrics` endpoint and uses a `ServiceMonitor` resource to expose them to Prometheus.
In order for this to work, you need to have `Prometheus Operator` installed in your cluster so that the `ServiceMonitor` resource can be created.

Use the following configuration to enable metrics:
```helm
testkube-cloud-api:
  prometheus:
    enabled: true
```

### Invitations

Testkube On-Prem allows you to invite users to Organizations and Environments within Testkube, granting them specific roles and permissions.

There are two supported invitation modes: `email` and `auto-accept`.
Use `email` to send an invitation for the user to accept, and `auto-accept` to automatically add users without requiring acceptance.

#### Invitations Via Email

If `testkube-cloud-api.api.inviteMode` is set to `email`, Testkube On-Prem will send emails when a user gets invited to
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

If `testkube-cloud-api.api.inviteMode` is set to `auto-accept`, Testkube On-Prem will automatically add users to
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

Testkube On-Prem supports integrating with existing infrastructure components such as MongoDB, NATS, Dex, etc.

### MongoDB

Testkube On-Prem uses MongoDB as a database for storing all the data.
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

Testkube On-Prem uses NATS as a message broker for communication between Control Plane components.

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

Testkube On-Prem uses MinIO as a storage backend for storing artifacts.

If you wish to use an existing MinIO instance, you can configure the following values:
```helm
testkube-cloud-api:
  minio:
    enabled: false
  api:
    minio: {} # check out the `testkube-cloud-api.api.minio` block in the values.yaml for all available settings
```

### Dex

Testkube On-Prem uses Dex as an identity provider.

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

1. Add our Testkube On-Prem Helm registry:
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
8. Install Testkube On-Prem as described in the [Installation](#installation) section

## FAQ

Q: Testkube Control Plane API is crashing (pod is in `Error`/`CrashLoopBackOff` state) with the following error:
```
panic: license file is invalid
```
A: Make sure the license file ends with a newline character.
There should be a new line after the `-----END LICENSE FILE-----` line in the license file.

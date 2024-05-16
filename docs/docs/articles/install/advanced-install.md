# Advanced Install

A variety of advanced topics to further customize your deployment.

## Organization Management

### Bootstrap Configuration

By default, Testkube will automatically add users to the default organizations when they get invited. You can change the bootstrap configuration to change this behaviour programmatically.

The simplest configuration is as follows. It creates a default org and environment and users will automatically join as admin:

```bash
testkube-cloud-api:
  api:
    features:
      bootstrapOrg: <your-org>
      bootstrapEnv: "Your first environment"
      bootstrapAdmin: <you@example.com>
```

You can use the full bootstrapConfig for more advanced settings:

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

Note: The default organization and environment mapping only apply on first sign in. After, you can remove users from environments or change roles through the Testkube UI.

Additionally, by default, Testkube Pro creates a personal organization for every new user. When using the default organization and environment configuration, you can turn off personal organizations using the following config:

```helm
testkube-cloud-api:
  api:
    features:
      disablePersonalOrgs: true
```

### Invitations

Users will now have to be invited within the dashboard. You can configure the SMTP server and Testkube will send e-mail invitations, alternatively new users will join the organisation without explicitly accepting the invitation.

```bash
testkube-cloud-api:
  inviteMode: `email`
  api:
    email:
      fromEmail: "no-reply@example.com"
      fromName: "Testkube On-prem"
    inviteMode: email
    smtp:
      host: <smtp host>
      port: <smtp port>
      username: <smtp username>
      password: <smtp password>
			# passwordSecretRef: <secret name>
```

## Custom Ingress Controller

By default, Testkube uses the NGINX Ingress Controller to ensure the reliable functioning of gRPC and Websockets protocols.

More specifically, these annotations are added to configre NGINX and should not be changed:

```bash
# gRPC Ingress:
annotations:
  nginx.ingress.kubernetes.io/proxy-body-size: 8m
  nginx.ingress.kubernetes.io/client-header-timeout: "10800"
  nginx.ingress.kubernetes.io/client-body-timeout: "10800"

# WebSockets ingress:
annotations:
  nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
  nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
```

To use your own ingress controller, reach out to our support team and we’ll gladly investigate your ingress of choice. Alternatively, you can give it a try yourself by deploying Testkube and seeing whether gRPC and WebSockets are working properly.

## Bring Your Own Infra

Testkube Enterprise supports integrating with existing infrastructure components such as MongoDB, NATS, Dex, etc. For production environments, it's recommended to use your own infra or to harden the sub-charts.

#### MongoDB

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

#### NATS

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

#### MinIO

Testkube Enterprise uses MinIO as a storage backend for storing artifacts.

If you wish to use an existing MinIO instance, you can configure the following values:

```helm
testkube-cloud-api:
  minio:
    enabled: false
  api:
    minio: {} # check out the `testkube-cloud-api.api.minio` block in the values.yaml for all available settings
```

#### Dex

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

## Air-gapped Environments

### Offline License

By default, Testkube will work with licenses that require internet connectivity. These licenses have the following format: `XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX-V3`. However, if you want to use Testkube in offline environments you will need to use an offline license.

[Contact support][contact] if you need an offline license.

Once you obtained an offline license, you should create a Shared Secret and afterwards

```bash
global:
  enterpriseOfflineAccess: true
  enterpriseLicenseSecretRef: testkube-enterprise-license
```

### Artifactory and Other Registry Proxies

By default, Testkube will pull images from the [docker.io](https://docker.io) registry. You can override the image of each individual service.

[contact]: https://testkube.io/contact

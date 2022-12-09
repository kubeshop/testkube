---
sidebar_position: 10
sidebar_label: Dependencies
---

# Dependencies for Testkube

Installing Testkube runs a Nats.io, Minio and MongoDB instance in your Kubernetes cluster. There is an option to replace some of these with your own instances.

## MongoDB

MongoDB is used for storage of Testkube Test results and various Testkube configurations as telemetry settings and cluster ID.

In order to use an external MongoDB instance, follow these steps:

1. Make sure you have access to the MongoDB you want to connect to - note: newest versions of MongoDB might not work optimally with Testkube, for the best experience, use MongoDB v4.4.12
2. Install testkube with --set mongo.enabled=false:
`kubectl testkube install --set mongo.enabled=false`
3. [Update MongoDB details for the api-server in the helm values with valid connection string](https://github.com/kubeshop/helm-charts/blob/main/charts/testkube/values.yaml)

### SSL connections

Inspecting the Testkube api-server manifest shows the following MongoDB-related environment variables:

* _"API_MONGO_DSN"_ (default:"mongodb://localhost:27017") - connection string
* _"API_MONGO_DB"_ (default:"testkube") - database name
* _"API_MONGO_SSL_CERT"_ (no default value) - reference to Kubernetes secret for MongoDB instances with SSL enabled

_API_MONGO_SSL_CERT_ expects the name of a Kubernetes secret containing all the necessary information to establish an SSL connection to the MongoDB instance. This secret has to be in the `testkube` namespace and should have the following structure:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mongo-ssl-certs
  namespace: testkube
type: Opaque
data:
  sslCertificateAuthorityFile: <base64 encoded root-ca.pem>
  sslClientCertificateKeyFile: <base64 encoded mongodb.pem>
  sslClientCertificateKeyFilePassword: <base64 encoded password>
```

To set this variable on helm-charts level, set [mongodb.sslCertSecret](https://github.com/kubeshop/helm-charts/blob/main/charts/testkube-api/values.yaml) to the name of the secret.

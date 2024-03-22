# Dependencies for Testkube

Installing Testkube runs a Nats.io, Minio and MongoDB instance in your Kubernetes cluster. There is an option to replace some of these with your own instances.

## MongoDB

MongoDB is used for storage of Testkube Test results and various Testkube configurations as telemetry settings and cluster ID.

In order to use an external MongoDB instance, follow these steps:

1. Make sure you have access to the MongoDB you want to connect to - note: newest versions of MongoDB might not work optimally with Testkube, for the best experience, use MongoDB v4.4.12
2. Install Testkube with --set mongo.enabled=false:
`kubectl testkube install --set mongo.enabled=false`
3. [Update MongoDB details for the api-server in the Helm values with valid connection string](https://github.com/kubeshop/helm-charts/blob/main/charts/testkube/values.yaml).

### SSL Connections

Inspecting the Testkube API-server manifest shows the following MongoDB-related environment variables:

* _"API_MONGO_DSN"_ (default:"mongodb://localhost:27017") - connection string
* _"API_MONGO_DB"_ (default:"testkube") - database name
* _"API_MONGO_SSL_CERT"_ (no default value) - reference to Kubernetes secret for MongoDB instances with SSL enabled
* _"API_MONGO_SSL_CA_FILE_KEY"_ (default:"sslCertificateAuthorityFile") - the key in the secret that marks the CA file
* _"API_MONGO_SSL_CLIENT_FILE_KEY"_ (default:"sslClientCertificateKeyFile") - the key in the secret that marks the client certificate file
* _"API_MONGO_SSL_CLIENT_FILE_PASS_KEY"_ (default:"sslClientCertificateKeyFilePassword") - the key in the secret that marks the client certificate file password

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

The keys of the fields can be modified. To set these variables on helm-charts level, set [mongodb.sslCertSecret](https://github.com/kubeshop/helm-charts/blob/main/charts/testkube-api/values.yaml) to the name of the secret. If needed, also set _mongodb.sslCAFileKey_, _mongo.sslClientFileKey_ and _mongodb.sslClientFilePassKey_.

### Amazon DocumentDB

Warning: DocumentDB will not be supported in future releases. This is compatible with older releases of Testkube. 

Testkube supports using [Amazon DocumentDB](https://aws.amazon.com/documentdb/), the managed version on MongoDB on AWS, as its database. Configuring it without TLS enabled is straightforward: add the connection string, and make sure the features that are not supported by DocumentDB are disabled. The parameters in the [helm-charts](https://github.com/kubeshop/helm-charts/blob/main/charts/testkube-api/values.yaml) are:

```bash
mongodb:
  dsn: "mongodb://docdbadmin:<insertYourPassword>@docdb.cluster.us-east-1.docdb.amazonaws.com:27017/?retryWrites=false"
  allowDiskUse: false
```

#### With TLS Enabled

Using DocumentDB with TLS enabled is fairly simple as well. You will need to specify the `dbType` and `allowTLS` in addition to the previous fields:

```bash
mongodb:
  dsn: "mongodb://docdbadmin:<insertYourPassword>@docdb.cluster.location.docdb.amazonaws.com:27017/?retryWrites=false"
  allowDiskUse: false
  dbType: docdb
  allowTLS: true
```

Testkube will download and use the CA certificates provided by AWS from https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem.

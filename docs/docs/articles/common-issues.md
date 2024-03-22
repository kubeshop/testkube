# Common Issues

## How do I expose Testkube to the Internet?

To expose Testkube to the Internet, you will need to create an Ingress for both the Testkube API and the Testkube dashboard.

Check the guides [here](./going-to-production.md) for different configurations.

## Access the Service Under Test (SUT) Using Testkube

- Services inside the same Kubernetes cluster can be accessed using the address `\<service-name\>.\<service-namespace\>.svc.cluster.local:\<port-number\>`. If there are network restrictions configured, Testkube will need permissions to access the SUT over the local network of the cluster.
- If Testkube and the SUT are not in the same cluster, SUT will have to be exposed to Testkube using an Ingress or a Load Balancer.

## If You're Still Having Issues

If these guides do not solve the issue that you encountered or you have other questions or comments, please contact us on [Slack](https://testkubeworkspace.slack.com/join/shared_invite/zt-2arhz5vmu-U2r3WZ69iPya5Fw0hMhRDg#/shared-invite/email).

## Other Installation Methods

### Installation on OpenShift Deployed on GCP

To install Testkube you need an empty OpenShift cluster. Once the cluster is up and running update `values.yaml` file, including the configuration below.

1. Add security context for MongoDB to `values.yaml`:

```yaml
mongodb:
  securityContext:
    enabled: true
    fsGroup: 1000650001
    runAsUser: 1000650001
  podSecurityContext:
    enabled: false
  containerSecurityContext:
    enabled: true
    runAsUser: 1000650001
    runAsNonRoot: true
  volumePermissions:
    enabled: false
  auth:
    enabled: false
```

2. Add security context for `Patch` and `Migrate` jobs that are a part of Testkube Operator configuration to `values.yaml`:

```yaml
testkube-operator:
  webhook:
    migrate:
      enabled: true
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]

    patch:
      enabled: true
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000650000
        fsGroup: 1000650000
```

3. Install Testkube specifying the path to the new `values.yaml` file

```
helm install testkube kubeshop/testkube --create-namespace --namespace testkube --values values.yaml
```

Please notice that since we've just installed MongoDB with a `testkube-mongodb` Helm release name, you are not required to reconfigure the Testkube API MongoDB connection URI. If you've installed with a different name/namespace, please adjust `--set testkube-api.mongodb.dsn: "mongodb://testkube-mongodb:27017"` to your MongoDB service.

### Installation with S3 Storage and IAM Authentication

To use S3 as storage, the steps are as follows:

1. Configure IAM role with the following permissions:

  s3:DeleteObject
  s3:GetObject
  s3:ListBucket
  s3:PutObject


2. Create a ServiceAccount with the ARN specified.
   e.g.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::265500248336:role/minio-example
  name: s3-access
  namespace: testkube
```
In the Helm values.yaml file:
3. Add the ARN annotation from above to `testkube-api.serviceAccount.annotations`.
4. Link the ServiceAccount to the `testkube-api.minio.serviceAccountName` and to `testkube-api.jobServiceAccountName`.
5. Leave `minio.minioRootUser`, `minio.minioRootPassword` and `storage.port` empty.
6. Set `storage.endpoint` to `s3.amazonaws.com`.

7. Install using Helm and the values file with the above modifications.

## Observability

There are two types of storage Mongo and Minio, read more details [here](./logging.md).

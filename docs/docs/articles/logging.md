# Logging

Testkube can be configured to use different storage for test logs output that can be specified in the Helm values.

```yaml
## Logs storage for Testkube API.
logs:
  ## where the logs should be stored there are 2 possible valuse : minio|mongo
  storage: "minio"
  ## if storage is set to minio then the bucket must be specified, if minio with s3 is used make sure to use a unique name
  bucket: "testkube-logs"
```

## [Mongo](https://www.mongodb.com/kubernetes)
When Mongo is specified, logs will be stored in a separate collection so the execution handling performance is not affected.

## [MinIO](https://min.io/)
When MinIO is specified, logs will be stored as separate files in the configured bucket of the MinIO instance or the S3 bucket if MinIO is configured to work with S3.
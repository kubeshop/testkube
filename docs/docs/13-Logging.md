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

## Mongo
When mongo is specified it will store in a separate collection so the execution handling performance is not affected.

## minIO
When minIO is specified, it will store the logs as separate files in the configured bucket of the minIO instance or S3 bucket if minIO is configured to work with S3.
## testkube upgrade

Upgrade Helm chart, install dependencies and run migrations

```
testkube upgrade [flags]
```

### Options

```
      --chart string       chart name (default "kubeshop/testkube")
  -h, --help               help for upgrade
      --name string        installation name (default "testkube")
      --namespace string   namespace where to install (default "testkube")
      --no-dashboard       don't install dashboard
      --no-minio           don't install MinIO
      --no-mongo           don't install MongoDB
      --values string      path to Helm values file
```

### Options inherited from parent commands

```
  -a, --api-uri string   api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string    client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --oauth-enabled    enable oauth (default true)
      --verbose          show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin


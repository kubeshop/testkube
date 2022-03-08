## testkube upgrade

Upgrade Helm chart, install dependencies and run migrations

```
testkube upgrade [flags]
```

### Options

```
      --chart string   chart name (default "kubeshop/testkube")
  -h, --help           help for upgrade
      --name string    installation name (default "testkube")
      --no-dashboard   don't install dashboard
      --no-jetstack    don't install Jetstack
      --no-minio       don't install MinIO
```

### Options inherited from parent commands

```
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             Show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin


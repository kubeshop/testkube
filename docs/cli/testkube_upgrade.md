## testkube upgrade

Upgrade Helm chart and run migrations

### Synopsis

Upgrade can be configured with use of particular 

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
      --analytics-enabled   should analytics be enabled (default true)
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - testkube entrypoint for plugin


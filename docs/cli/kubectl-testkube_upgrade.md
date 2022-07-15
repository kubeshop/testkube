## kubectl-testkube upgrade

Upgrade Helm chart, install dependencies and run migrations

```
kubectl-testkube upgrade [flags]
```

### Options

```
      --chart string    chart name (default "kubeshop/testkube")
  -h, --help            help for upgrade
      --name string     installation name (default "testkube")
      --no-dashboard    don't install dashboard
      --no-jetstack     don't install Jetstack
      --no-minio        don't install MinIO
      --no-mongo        don't install MongoDB
      --values string   path to Helm values file
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin


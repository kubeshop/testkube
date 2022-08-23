## kubectl-testkube status telemetry

Get telemetry status

```
kubectl-testkube status telemetry [flags]
```

### Options

```
  -h, --help   help for telemetry
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data (default true)
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube status](kubectl-testkube_status.md)	 - Show status of feature or resource


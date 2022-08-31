## kubectl-testkube download artifacts

download artifacts

```
kubectl-testkube download artifacts <executionID> [flags]
```

### Options

```
      --download-dir string   download dir (default "artifacts")
  -e, --execution-id string   ID of the execution
  -h, --help                  help for artifacts
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data
      --verbose             should I show additional debug messages
```

### SEE ALSO

* [kubectl-testkube download](kubectl-testkube_download.md)	 - Artifacts management commands


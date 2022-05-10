## kubectl-testkube download artifact

download artifact

```
kubectl-testkube download artifact <executionID> <fileName> <destinationDir> [flags]
```

### Options

```
  -d, --destination string    name of the file
  -e, --execution-id string   ID of the execution
  -f, --filename string       name of the file
  -h, --help                  help for artifact
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics (default true)
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --verbose             should I show additional debug messages
```

### SEE ALSO

* [kubectl-testkube download](kubectl-testkube_download.md)	 - Artifacts management commands


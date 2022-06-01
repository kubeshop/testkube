## kubectl-testkube create executor

Create new Executor

### Synopsis

Create new Executor Custom Resource

```
kubectl-testkube create executor [flags]
```

### Options

```
      --crd-only               generate only executor crd
      --executor-type string   executor type (defaults to job) (default "job")
  -h, --help                   help for executor
  -i, --image string           if uri is git repository we can set additional branch parameter
  -j, --job-template string    if executor needs to be launched using custom job specification
  -l, --label stringToString   label key value pair: --label key1=value1 (default [])
  -n, --name string            unique test name - mandatory
  -t, --types stringArray      types handled by executor
  -u, --uri string             if resource need to be loaded from URI
```

### Options inherited from Parent Commands

```
      --analytics-enabled   enable analytics
  -a, --api-uri string      api uri, default value read from config if set
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube create](kubectl-testkube_create.md)	 - Create resource


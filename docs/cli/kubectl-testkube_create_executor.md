## kubectl-testkube create executor

Create new Executor

### Synopsis

Create new Executor Custom Resource

```
kubectl-testkube create executor [flags]
```

### Options

```
      --executor-type string   executor type (defaults to job) (default "job")
  -h, --help                   help for executor
  -i, --image string           if uri is git repository we can set additional branch parameter
  -j, --job-template string    if executor needs to be launched using custom job specification
  -n, --name string            unique test name - mandatory
  -t, --types stringArray      types handled by executor
  -u, --uri string             if resource need to be loaded from URI
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics (default true)
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
  -v, --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube create](kubectl-testkube_create.md)	 - Create resource


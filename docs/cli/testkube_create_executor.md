## testkube create executor

Create new Executor

### Synopsis

Create new Executor Custom Resource

```
testkube create executor [flags]
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
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             Show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource


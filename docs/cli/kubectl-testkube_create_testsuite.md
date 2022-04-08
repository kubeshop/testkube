## kubectl-testkube create testsuite

Create new TestSuite

### Synopsis

Create new TestSuite Custom Resource

```
kubectl-testkube create testsuite [flags]
```

### Options

```
  -f, --file string            JSON test suite file - will be read from stdin if not specified, look at testkube.TestUpsertRequest
  -h, --help                   help for testsuite
  -l, --label stringToString   label key value pair: --label key1=value1 (default [])
      --name string            Set/Override test suite name
      --schedule string        test suite schedule in a cronjob form: * * * * *
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


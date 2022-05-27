## kubectl-testkube update testsuite

Update Test Suite

### Synopsis

Update Test Custom Resource Definitions, 

```
kubectl-testkube update testsuite [flags]
```

### Options

```
  -f, --file string            JSON test file - will be read from stdin if not specified, look at testkube.TestUpsertRequest
  -h, --help                   help for testsuite
  -l, --label stringToString   label key value pair: --label key1=value1 (default [])
      --name string            Set/Override test suite name
      --schedule string        test suite schedule in a cronjob form: * * * * *
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

* [kubectl-testkube update](kubectl-testkube_update.md)	 - Update resource


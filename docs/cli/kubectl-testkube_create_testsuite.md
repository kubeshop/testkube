## kubectl-testkube create testsuite

Create new TestSuite

### Synopsis

Create new TestSuite Custom Resource

```
kubectl-testkube create testsuite [flags]
```

### Options

```
  -f, --file string                      JSON test suite file - will be read from stdin if not specified, look at testkube.TestUpsertRequest
  -h, --help                             help for testsuite
  -l, --label stringToString             label key value pair: --label key1=value1 (default [])
      --name string                      Set/Override test suite name
      --schedule string                  test suite schedule in a cronjob form: * * * * *
  -s, --secret-variable stringToString   secret variable key value pair: --secret-variable key1=value1 (default [])
  -v, --variable stringToString          param key value pair: --variable key1=value1 (default [])
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -a, --api-uri string      api uri, default value read from config if set
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --crd-only            generate only crd
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube create](kubectl-testkube_create.md)	 - Create resource


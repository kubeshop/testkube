## kubectl-testkube delete testsuite

Delete test suite

### Synopsis

Delete test suite by name

```
kubectl-testkube delete testsuite <testSuiteName> [flags]
```

### Options

```
      --all             Delete all tests
  -h, --help            help for testsuite
  -l, --label strings   label key value pair: --label key1=value1
```

### Options inherited from Parent Commands

```
      --analytics-enabled   enable analytics
  -a, --api-uri string      api uri, default value read from config if set
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --namespace string    kubernetes namespace (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             should I show additional debug messages
```

### SEE ALSO

* [kubectl-testkube delete](kubectl-testkube_delete.md)	 - Delete resources


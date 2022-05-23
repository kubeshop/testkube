## kubectl-testkube delete test

Delete Test

```
kubectl-testkube delete test [testName] [flags]
```

### Options

```
      --all             Delete all tests
  -h, --help            help for test
  -l, --label strings   label key value pair: --label key1=value1
```

### Options inherited from parent commands

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


## kubectl-testkube update

Update resource

```
kubectl-testkube update <resourceName> [flags]
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube update executor](kubectl-testkube_update_executor.md)	 - Update Executor
* [kubectl-testkube update test](kubectl-testkube_update_test.md)	 - Update test
* [kubectl-testkube update testsource](kubectl-testkube_update_testsource.md)	 - Update TestSource
* [kubectl-testkube update testsuite](kubectl-testkube_update_testsuite.md)	 - Update Test Suite


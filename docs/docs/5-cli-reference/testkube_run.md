## kubectl-testkube run

Runs tests or test suites

```
kubectl-testkube run <resourceName> [flags]
```

### Options

```
  -h, --help   help for run
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
* [kubectl-testkube run test](kubectl-testkube_run_test.md)	 - Starts new test
* [kubectl-testkube run testsuite](kubectl-testkube_run_testsuite.md)	 - Starts new test suite


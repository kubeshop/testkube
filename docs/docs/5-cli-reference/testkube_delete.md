## kubectl-testkube delete

Delete resources

```
kubectl-testkube delete <resourceName> [flags]
```

### Options

```
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -h, --help               help for delete
      --namespace string   kubernetes namespace (default "testkube")
      --verbose            should I show additional debug messages
```

### Options inherited from parent commands

```
  -a, --api-uri string   api uri, default value read from config if set (default "http://localhost:8088")
      --oauth-enabled    enable oauth (default true)
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube delete executor](kubectl-testkube_delete_executor.md)	 - Delete Executor
* [kubectl-testkube delete test](kubectl-testkube_delete_test.md)	 - Delete Test
* [kubectl-testkube delete testsource](kubectl-testkube_delete_testsource.md)	 - Delete test source
* [kubectl-testkube delete testsuite](kubectl-testkube_delete_testsuite.md)	 - Delete test suite
* [kubectl-testkube delete webhook](kubectl-testkube_delete_webhook.md)	 - Delete webhook


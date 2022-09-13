## kubectl-testkube delete

Delete resources

```
kubectl-testkube delete <resourceName> [flags]
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube delete executor](kubectl-testkube_delete_executor.md)	 - Delete Executor
* [kubectl-testkube delete test](kubectl-testkube_delete_test.md)	 - Delete Test
* [kubectl-testkube delete testsuite](kubectl-testkube_delete_testsuite.md)	 - Delete test suite
* [kubectl-testkube delete webhook](kubectl-testkube_delete_webhook.md)	 - Delete webhook


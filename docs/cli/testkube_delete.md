## testkube delete

Delete resources

```
testkube delete <resourceName> [flags]
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

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube delete executor](testkube_delete_executor.md)	 - Delete Executor
* [testkube delete test](testkube_delete_test.md)	 - Delete Test
* [testkube delete testsuite](testkube_delete_testsuite.md)	 - Delete test suite
* [testkube delete webhook](testkube_delete_webhook.md)	 - Delete webhook


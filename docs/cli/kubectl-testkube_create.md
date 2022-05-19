## kubectl-testkube create

Create resource

```
kubectl-testkube create <resourceName> [flags]
```

### Options

```
  -h, --help   help for create
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set (default "http://testdash.testkube.io/api")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth (default true)
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube create executor](kubectl-testkube_create_executor.md)	 - Create new Executor
* [kubectl-testkube create test](kubectl-testkube_create_test.md)	 - Create new Test
* [kubectl-testkube create testsuite](kubectl-testkube_create_testsuite.md)	 - Create new TestSuite
* [kubectl-testkube create webhook](kubectl-testkube_create_webhook.md)	 - Create new Webhook


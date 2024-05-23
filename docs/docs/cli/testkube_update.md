## testkube update

Update resource

```
testkube update <resourceName> [flags]
```

### Options

```
  -h, --help   help for update
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube update executor](testkube_update_executor.md)	 - Update Executor
* [testkube update template](testkube_update_template.md)	 - Update Template
* [testkube update test](testkube_update_test.md)	 - Update test
* [testkube update testsource](testkube_update_testsource.md)	 - Update TestSource
* [testkube update testsuite](testkube_update_testsuite.md)	 - Update Test Suite
* [testkube update webhook](testkube_update_webhook.md)	 - Update Webhook


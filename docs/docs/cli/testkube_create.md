## testkube create

Create resource

```
testkube create <resourceName> [flags]
```

### Options

```
      --crd-only   generate only crd
  -h, --help       help for create
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
* [testkube create executor](testkube_create_executor.md)	 - Create new Executor
* [testkube create template](testkube_create_template.md)	 - Create a new Template.
* [testkube create test](testkube_create_test.md)	 - Create new Test
* [testkube create testsource](testkube_create_testsource.md)	 - Create new TestSource
* [testkube create testsuite](testkube_create_testsuite.md)	 - Create new TestSuite
* [testkube create testworkflow](testkube_create_testworkflow.md)	 - Create test workflow
* [testkube create testworkflowtemplate](testkube_create_testworkflowtemplate.md)	 - Create test workflow template
* [testkube create webhook](testkube_create_webhook.md)	 - Create new Webhook


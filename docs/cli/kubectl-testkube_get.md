## kubectl-testkube get

Get resources

### Synopsis

Get available resources, get single item or list

```
kubectl-testkube get <resourceName> [flags]
```

### Options

```
      --go-template string   go template to render (default "{{.}}")
  -h, --help                 help for get
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube get artifact](kubectl-testkube_get_artifact.md)	 - List artifacts of the given execution ID
* [kubectl-testkube get execution](kubectl-testkube_get_execution.md)	 - Lists or gets test executions
* [kubectl-testkube get executor](kubectl-testkube_get_executor.md)	 - Gets executor details
* [kubectl-testkube get test](kubectl-testkube_get_test.md)	 - Get all available tests
* [kubectl-testkube get testsuite](kubectl-testkube_get_testsuite.md)	 - Get test suite by name
* [kubectl-testkube get testsuiteexecution](kubectl-testkube_get_testsuiteexecution.md)	 - Gets TestSuite Execution details
* [kubectl-testkube get webhook](kubectl-testkube_get_webhook.md)	 - Get webhook details


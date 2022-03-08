## testkube get

Get resources

### Synopsis

Get available resources, get single item or list

```
testkube get <resourceName> [flags]
```

### Options

```
      --go-template string   go template to render (default "{{.}}")
  -h, --help                 help for get
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
```

### Options inherited from parent commands

```
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             Show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube get artifact](testkube_get_artifact.md)	 - List artifacts of the given execution ID
* [testkube get execution](testkube_get_execution.md)	 - Lists or gets test executions
* [testkube get executor](testkube_get_executor.md)	 - Gets executor details
* [testkube get test](testkube_get_test.md)	 - Get all available tests
* [testkube get testsuite](testkube_get_testsuite.md)	 - Get test suite by name
* [testkube get testuiteexecution](testkube_get_testuiteexecution.md)	 - Gets TestSuite Execution details
* [testkube get webhook](testkube_get_webhook.md)	 - Get webhook details


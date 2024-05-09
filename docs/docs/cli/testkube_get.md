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
  -o, --output string        output type can be one of json|yaml|pretty|go (default "pretty")
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
* [testkube get artifact](testkube_get_artifact.md)	 - List artifacts of the given test, test suite or test workflow execution name
* [testkube get context](testkube_get_context.md)	 - Set context for Testkube Pro
* [testkube get execution](testkube_get_execution.md)	 - Lists or gets test executions
* [testkube get executor](testkube_get_executor.md)	 - Gets executor details
* [testkube get template](testkube_get_template.md)	 - Get template details.
* [testkube get test](testkube_get_test.md)	 - Get all available tests
* [testkube get testsource](testkube_get_testsource.md)	 - Get test source details
* [testkube get testsuite](testkube_get_testsuite.md)	 - Get test suite by name
* [testkube get testsuiteexecution](testkube_get_testsuiteexecution.md)	 - Gets TestSuite Execution details
* [testkube get testworkflow](testkube_get_testworkflow.md)	 - Get all available test workflows
* [testkube get testworkflowexecution](testkube_get_testworkflowexecution.md)	 - Gets TestWorkflow execution details
* [testkube get testworkflowtemplate](testkube_get_testworkflowtemplate.md)	 - Get all available test workflow templates
* [testkube get webhook](testkube_get_webhook.md)	 - Get webhook details


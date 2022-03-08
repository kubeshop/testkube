## testkube get testuiteexecution

Gets TestSuite Execution details

### Synopsis

Gets TestSuite Execution details by ID, or list if id is not passed

```
testkube get testuiteexecution [executionID] [flags]
```

### Options

```
  -h, --help                help for testuiteexecution
  -l, --label strings       label key value pair: --label key1=value1
      --limit int           max number of records to return (default 1000)
      --test-suite string   test suite name
```

### Options inherited from parent commands

```
      --analytics-enabled    Enable analytics (default true)
  -c, --client string        Client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --go-template string   go template to render (default "{{.}}")
  -s, --namespace string     Kubernetes namespace (default "testkube")
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
  -v, --verbose              Show additional debug messages
```

### SEE ALSO

* [testkube get](testkube_get.md)	 - Get resources


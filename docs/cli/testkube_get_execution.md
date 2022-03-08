## testkube get execution

Lists or gets test executions

### Synopsis

Getting list of execution for given test name or recent executions if there is no test name passed

```
testkube get execution [executionID] [flags]
```

### Options

```
  -h, --help            help for execution
  -l, --label strings   label key value pair: --label key1=value1
      --limit int       records limit (default 10)
      --test string     test id
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


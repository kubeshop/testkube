## testkube get execution

Lists or gets test executions

### Synopsis

Getting list of execution for given test name or recent executions if there is no test name passed

```
testkube get execution [executionID][executionName] [flags]
```

### Options

```
  -h, --help            help for execution
  -l, --label strings   label key value pair: --label key1=value1
      --limit int       records limit (default 10)
      --logs-only       show only execution logs
      --test string     test id
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --go-template string      go template to render (default "{{.}}")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
  -o, --output string           output type can be one of json|yaml|pretty|go (default "pretty")
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube get](testkube_get.md)	 - Get resources


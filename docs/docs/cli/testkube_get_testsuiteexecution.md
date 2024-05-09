## testkube get testsuiteexecution

Gets TestSuite Execution details

### Synopsis

Gets TestSuite Execution details by ID, or list if id is not passed

```
testkube get testsuiteexecution [executionID] [flags]
```

### Options

```
  -h, --help                help for testsuiteexecution
  -l, --label strings       label key value pair: --label key1=value1
      --limit int           max number of records to return (default 1000)
      --test-suite string   test suite name
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


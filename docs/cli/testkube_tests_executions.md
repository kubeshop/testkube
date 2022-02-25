## testkube tests executions

List test executions

### Synopsis

Getting list of execution for given test name or recent executions if there is no test name passed

```
testkube tests executions [testName] [flags]
```

### Options

```
  -h, --help           help for executions
      --tags strings   comma separated list of tags: --tags tag1,tag2,tag3
```

### Options inherited from parent commands

```
      --analytics-enabled    should analytics be enabled (default true)
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "testkube")
  -o, --output string        output type one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube tests](testkube_tests.md)	 - Tests management commands


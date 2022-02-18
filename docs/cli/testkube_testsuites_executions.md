## testkube testsuites executions

Gets tests executions list

### Synopsis

Gets tests executions list, can be filtered by test name

```
testkube testsuites executions [testSuiteName] [flags]
```

### Options

```
  -h, --help           help for executions
      --limit int      max number of records to return (default 1000)
      --tags strings   comma separated list of tags: --tags tag1,tag2,tag3
```

### Options inherited from parent commands

```
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -s, --namespace string     kubernetes namespace (default "testkube")
  -o, --output string        output type one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```

### SEE ALSO

* [testkube testsuites](testkube_testsuites.md)	 - Test suites management commands


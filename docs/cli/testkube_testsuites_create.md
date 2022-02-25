## testkube testsuites create

Create new test suite

### Synopsis

Create new TestSuite Custom Resource

```
testkube testsuites create [flags]
```

### Options

```
  -f, --file string    JSON test suite file - will be read from stdin if not specified, look at testkube.TestUpsertRequest
  -h, --help           help for create
      --name string    Set/Override test suite name
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

* [testkube testsuites](testkube_testsuites.md)	 - Test suites management commands


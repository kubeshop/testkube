## testkube create testsuite

Create new TestSuite

### Synopsis

Create new TestSuite Custom Resource

```
testkube create testsuite [flags]
```

### Options

```
  -f, --file string            JSON test suite file - will be read from stdin if not specified, look at testkube.TestUpsertRequest
  -h, --help                   help for testsuite
  -l, --label stringToString   label key value pair: --label key1=value1 (default [])
      --name string            Set/Override test suite name
```

### Options inherited from parent commands

```
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             Show additional debug messages
```

### SEE ALSO

* [testkube create](testkube_create.md)	 - Create resource


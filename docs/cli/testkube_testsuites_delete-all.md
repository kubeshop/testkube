## testkube testsuites delete-all

Delete all tests in namespace

```
testkube testsuites delete-all [flags]
```

### Options

```
  -h, --help   help for delete-all
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


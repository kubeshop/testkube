## testkube testsuites

Test suites management commands

### Synopsis

All available test suites and test suite executions commands

```
testkube testsuites [flags]
```

### Options

```
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -h, --help                 help for testsuites
  -o, --output string        output type one of raw|json|go  (default "raw")
```

### Options inherited from parent commands

```
      --analytics-enabled   should analytics be enabled (default true)
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - testkube entrypoint for plugin
* [testkube testsuites create](testkube_testsuites_create.md)	 - Create new test suite
* [testkube testsuites delete](testkube_testsuites_delete.md)	 - Delete test suite
* [testkube testsuites delete-all](testkube_testsuites_delete-all.md)	 - Delete all test suites in namespace
* [testkube testsuites execution](testkube_testsuites_execution.md)	 - Gets test suite execution details
* [testkube testsuites executions](testkube_testsuites_executions.md)	 - Gets test suites executions list
* [testkube testsuites get](testkube_testsuites_get.md)	 - Get test suite by name
* [testkube testsuites list](testkube_testsuites_list.md)	 - Get all available test suites
* [testkube testsuites run](testkube_testsuites_run.md)	 - Starts new test suite
* [testkube testsuites update](testkube_testsuites_update.md)	 - Update Test Suite
* [testkube testsuites watch](testkube_testsuites_watch.md)	 - Watch test


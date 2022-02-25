## testkube tests

Tests management commands

### Synopsis

All available tests and test executions commands

```
testkube tests [flags]
```

### Options

```
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -h, --help                 help for tests
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
* [testkube tests abort](testkube_tests_abort.md)	 - Aborts execution of the test
* [testkube tests create](testkube_tests_create.md)	 - Create new test
* [testkube tests delete](testkube_tests_delete.md)	 - Delete tests
* [testkube tests delete-all](testkube_tests_delete-all.md)	 - Delete all tests
* [testkube tests execution](testkube_tests_execution.md)	 - Gets test execution details
* [testkube tests executions](testkube_tests_executions.md)	 - List test executions
* [testkube tests get](testkube_tests_get.md)	 - Get test by name
* [testkube tests list](testkube_tests_list.md)	 - Get all available tests
* [testkube tests run](testkube_tests_run.md)	 - Starts new test
* [testkube tests update](testkube_tests_update.md)	 - Update test
* [testkube tests watch](testkube_tests_watch.md)	 - Watch logs output from executor pod


# Testkube Tests

Tests management commands.

## **Synopsis**

All available tests and test executions commands:

```
testkube tests [flags]
```

## **Options**

```
      --go-template string   When choosing output==go. pass golang template (default "{{ . | printf \"%+v\"  }}").
  -h, --help                 Help for tests.
  -o, --output string        Output type. Either raw, json or go  (default "raw").
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled   should analytics be enabled (default true)
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    kubernetes namespace (default "testkube")
  -v, --verbose             should I show additional debug messages
```

## **SEE ALSO**

* [Testkube](testkube.md)	 - Testkube entrypoint for plugins.
* [Testkube Tests Abort](testkube_tests_abort.md)	 - Abort execution of the test.
* [Testkube Tests Create](testkube_tests_create.md)	 - Create a new test.
* [Testkube Tests Delete](testkube_tests_delete.md)	 - Delete tests.
* [Testkube Tests Delete-all](testkube_tests_delete-all.md)	 - Delete all tests.
* [Testkube Tests Execution](testkube_tests_execution.md)	 - Get test execution details.
* [Testkube Tests Executions](testkube_tests_executions.md)	 - List test executions.
* [Testkube Tests Get](testkube_tests_get.md)	 - Get test by name.
* [Testkube Tests List](testkube_tests_list.md)	 - Get all available tests.
* [Testkube Tests Run](testkube_tests_run.md)	 - Start a new test.
* [Testkube Tests Update](testkube_tests_update.md)	 - Update a test.
* [Testkube Tests Watch](testkube_tests_watch.md)	 - Watch logs output from the executor pod.


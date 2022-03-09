# Testkube Testsuites

Testsuite management commands:

## **Synopsis**

All available testsuites and test suite executions commands:

```
testkube testsuites [flags]
```

## **Options**

```
      --go-template string   When choosing output==go, pass golang template (default "{{ . | printf \"%+v\"  }}").
  -h, --help                 Help for testsuites.
  -o, --output string        Output type. Either raw, json or go  (default "raw").
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled   Enable analytics (default "true").
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy").
  -s, --namespace string    Kubernetes namespace (default "testkube").
  -v, --verbose             Show additional debug messages.
```

## **SEE ALSO**

* [Testkube](testkube.md)	 - Testkube entrypoint for plugins.
* [Testkube Test Suites Create](testkube_testsuites_create.md)	 - Create new test suites.
* [Testkube Test Suites Delete](testkube_testsuites_delete.md)	 - Delete test suites.
* [Testkube Test Suites Delete-all](testkube_testsuites_delete-all.md)	 - Delete all test suites in namespace.
* [Testkube Test Suites Execution](testkube_testsuites_execution.md)	 - Get test suite execution details.
* [Testkube Test Suites Executions](testkube_testsuites_executions.md)	 - Get test suites executions list.
* [Testkube Test Suites Get](testkube_testsuites_get.md)	 - Get test suite by name.
* [Testkube Test Suites List](testkube_testsuites_list.md)	 - Get all available test suites.
* [Testkube Test Suites Run](testkube_testsuites_run.md)	 - Starts a new test suite.
* [Testkube Test Suites Update](testkube_testsuites_update.md)	 - Update a test suite.
* [Testkube Test Suites Watch](testkube_testsuites_watch.md)	 - Watch test suites.


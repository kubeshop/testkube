# Testkube Tests Executions

## **Synopsis**

Provides a list of executions for a given test name. If no test name is passed, a recent list of test executions will be returned.

```
testkube tests executions [testName] [flags]
```

## **Options**

```
  -h, --help           Help for executions.
      --tags strings   A comma separated list of tags: --tags tag1,tag2,tag3.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled    Enable analytics (default "true").
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy").
      --go-template string   When choosing output==go, pass golang template (default "{{ . | printf \"%+v\"  }}").
  -s, --namespace string     Kubernetes namespace (default "testkube").
  -o, --output string        Output type - raw, json or go  (default "raw").
  -v, --verbose              Show additional debug messages.
```

## **SEE ALSO**

* [Testkube Tests](testkube_tests.md)	 - Tests management commands.


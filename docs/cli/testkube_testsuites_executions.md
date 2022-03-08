# Testkube Testsuites Executions

## **Synopsis**

Get a Test Suites executions list, can be filtered by test name:

```
testkube testsuites executions [testSuiteName] [flags]
```

## **Options**

```
  -h, --help           Help for executions
      --limit int      Max number of records to return (default 1000).
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

* [Testkube Testsuites](testkube_testsuites.md)	 - Testsuites management commands.


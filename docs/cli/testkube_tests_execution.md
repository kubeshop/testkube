# Testkube Tests Execution

## **Synopsis**

Gets test execution details. The output format can be changed.

```
testkube tests execution <executionID> [flags]
```

## **Options**

```
  -h, --help   Help for execution.
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


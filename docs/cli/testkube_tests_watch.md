# Testkube Tests Watch

Watch logs output from executor pod.

## **Synopsis**

Gets test execution details until the return of **success** or **error** and blocks until the state is complete.

```
testkube tests watch <executionID> [flags]
```

## **Options**

```
  -h, --help   Help for watch.
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


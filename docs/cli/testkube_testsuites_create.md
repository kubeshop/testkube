# Testkube Testsuites Create

## **Synopsis**

Create a new Test Suite Custom Resource:

```
testkube testsuites create [flags]
```

## **Options**

```
  -f, --file string    JSON test suite file - will be read from stdin. If not specified, look at testkube.TestUpsertRequest.
  -h, --help           Help for create.
      --name string    Set/Override testsuite name.
      --tags strings   Comma separated list of tags: --tags tag1,tag2,tag3.
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


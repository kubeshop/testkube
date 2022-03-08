# Testkube Testsuites Delete-all

Delete all Test Suites in a namespace:

```
testkube testsuites delete-all [flags]
```

## **Options**

```
  -h, --help   Help for delete-all.
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


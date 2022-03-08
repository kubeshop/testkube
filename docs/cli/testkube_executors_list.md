# Testkube Executors List

## **Synopsis**

Gets a list of executors. The output format can be changed.

```
testkube executors list [flags]
```

## **Options**

```
  -h, --help   Help for list.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled    Enable analytics (default "true").
  -c, --client string        Client used for connecting to testkube API one of proxy|direct (default "proxy").
      --go-template string   When choosing output==go, pass golang template (default "{{ . | printf \"%+v\"  }}").
  -s, --namespace string     Kubernetes namespace (default "testkube").
  -o, --output string        Output type - raw, json or go  (default "raw").
  -v, --verbose              Show additional debug messagesl
```

## **SEE ALSO**

* [Testkube Executors](testkube_executors.md)	 - Executor management commands.


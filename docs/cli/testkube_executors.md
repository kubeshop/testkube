# Testkube Executors

Executor management commands.

```
testkube executors [flags]
```

## **Options**

```
      --go-template string   When choosing output==go, pass golang template (default "{{ . | printf \"%+v\"  }}").
  -h, --help                 Help for executors.
  -o, --output string        Output type - raw, json or go  (default "raw").
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
* [Testkube Executors Create](testkube_executors_create.md)	 - Create a new executor.
* [Testkube Executors Delete](testkube_executors_delete.md)	 - Delete executor.
* [Testkube Executors Get](testkube_executors_get.md)	 - Get executor details.
* [Testkube Executors List](testkube_executors_list.md)	 - Get a list of executors.


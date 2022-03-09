# Testkube Executors Create

Create a new Executor.

## **Synopsis**

Create a new Executor Custom Resource:

```
testkube executors create [flags]
```

## **Options**

```
      --executor-type string   Executor type (default "job").
  -h, --help                   Help for create.
  -i, --image string           If uri is a Git repository, set additional branch parameter.
  -n, --name string            Unique test name - mandatory.
  -t, --types stringArray      Types handled by exeutor.
  -u, --uri string             Load resource from URI.
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

* [Testkube Executors](testkube_executors.md)	 - Executor management commands.


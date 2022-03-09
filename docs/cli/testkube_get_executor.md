## testkube get executor

Gets executor details

### Synopsis

Gets executor, you can change output format

```
testkube get executor [executorName] [flags]
```

### Options

```
  -h, --help   help for executor
```

### Options inherited from parent commands

```
      --analytics-enabled    Enable analytics (default true)
  -c, --client string        Client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --go-template string   go template to render (default "{{.}}")
  -s, --namespace string     Kubernetes namespace (default "testkube")
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
  -v, --verbose              Show additional debug messages
```

### SEE ALSO

* [testkube get](testkube_get.md)	 - Get resources


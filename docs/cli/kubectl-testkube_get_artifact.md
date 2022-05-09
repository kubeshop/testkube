## kubectl-testkube get artifact

List artifacts of the given execution ID

```
kubectl-testkube get artifact <executionID> [flags]
```

### Options

```
  -e, --execution-id string   ID of the execution
  -h, --help                  help for artifact
```

### Options inherited from parent commands

```
      --analytics-enabled    enable analytics
  -c, --client string        client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --go-template string   go template to render (default "{{.}}")
  -s, --namespace string     Kubernetes namespace, default value read from config if set (default "testkube")
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
  -v, --verbose              show additional debug messages
```

### SEE ALSO

* [kubectl-testkube get](kubectl-testkube_get.md)	 - Get resources


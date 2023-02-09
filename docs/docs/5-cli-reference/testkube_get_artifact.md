## testkube get artifact

List artifacts of the given test or test suite execution name

```
testkube get artifact <executionName> [flags]
```

### Options

```
  -e, --execution-id string   ID of the execution
  -h, --help                  help for artifact
```

### Options inherited from parent commands

```
  -a, --api-uri string       api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string        client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --go-template string   go template to render (default "{{.}}")
      --namespace string     Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled        enable oauth (default true)
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
      --verbose              show additional debug messages
```

### SEE ALSO

* [testkube get](testkube_get.md)	 - Get resources


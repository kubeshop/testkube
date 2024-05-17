## testkube get template

Get template details.

### Synopsis

Get template allows you to change the output format. To get single details, pass the template name as the first argument.

```
testkube get template <templateName> [flags]
```

### Options

```
      --crd-only        show only test crd
  -h, --help            help for template
  -l, --label strings   label key value pair: --label key1=value1
  -n, --name string     unique template name, you can also pass it as argument
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --go-template string      go template to render (default "{{.}}")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
  -o, --output string           output type can be one of json|yaml|pretty|go (default "pretty")
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube get](testkube_get.md)	 - Get resources


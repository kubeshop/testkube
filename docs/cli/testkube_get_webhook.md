## testkube get webhook

Get webhook details

### Synopsis

Get webhook, you can change output format, to get single details pass name as first arg

```
testkube get webhook <webhookName> [flags]
```

### Options

```
  -h, --help          help for webhook
  -n, --name string   unique webhook name, you can also pass it as argument
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


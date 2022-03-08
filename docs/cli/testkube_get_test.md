## testkube get test

Get all available tests

### Synopsis

Getting all available tests from given namespace - if no namespace given "testkube" namespace is used

```
testkube get test <testName> [flags]
```

### Options

```
  -h, --help            help for test
  -l, --label strings   label key value pair: --label key1=value1
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


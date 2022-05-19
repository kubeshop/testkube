## kubectl-testkube get testsuite

Get test suite by name

### Synopsis

Getting test suite from given namespace - if no namespace given "testkube" namespace is used

```
kubectl-testkube get testsuite <testSuiteName> [flags]
```

### Options

```
  -h, --help            help for testsuite
  -l, --label strings   label key value pair: --label key1=value1
      --no-execution    don't show latest execution
```

### Options inherited from parent commands

```
      --analytics-enabled    enable analytics
  -w, --api-uri string       api uri, default value read from config if set (default "http://testdash.testkube.io/api")
  -c, --client string        client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --go-template string   go template to render (default "{{.}}")
      --namespace string     Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled        enable oauth (default true)
  -o, --output string        output type can be one of json|yaml|pretty|go-template (default "pretty")
      --verbose              show additional debug messages
```

### SEE ALSO

* [kubectl-testkube get](kubectl-testkube_get.md)	 - Get resources


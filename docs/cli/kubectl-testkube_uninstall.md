## kubectl-testkube uninstall

Uninstall Helm chart registry in current kubectl context

### Synopsis

Uninstall Helm chart registry in current kubectl context

```
kubectl-testkube uninstall [flags]
```

### Options

```
  -h, --help          help for uninstall
      --name string   installation name (default "testkube")
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set (default "http://testdash.testkube.io/api")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth (default true)
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin


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
      --analytics-enabled   enable analytics (default true)
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
  -v, --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin


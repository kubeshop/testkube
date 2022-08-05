## kubectl-testkube debug

Print environment information for debugging

### Synopsis

Debug Testkube

```
  kubectl-testkube debug [flags]
  kubectl-testkube debug [command]
```

### Options

```
  -h, --help                   help for debug
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)  - Testkube entrypoint for kubectl plugin
* [kubectl-testkube debug info](kubectl-testkube_debug_info.md) - Show debug info

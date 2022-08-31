## kubectl-testkube enable

Enable feature

```
kubectl-testkube enable <feature> [flags]
```

### Options

```
  -h, --help   help for enable
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

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube enable oauth](kubectl-testkube_enable_oauth.md)	 - enable oauth authentication for direct api
* [kubectl-testkube enable telemetry](kubectl-testkube_enable_telemetry.md)	 - Enable collecting of anonymous telemetry data


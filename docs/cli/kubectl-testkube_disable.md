## kubectl-testkube disable

Disable feature

```
kubectl-testkube disable <feature> [flags]
```

### Options

```
  -h, --help   help for disable
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
* [kubectl-testkube disable oauth](kubectl-testkube_disable_oauth.md)	 - disable oauth authentication for direct api
* [kubectl-testkube disable telemetry](kubectl-testkube_disable_telemetry.md)	 - disable collecting of anonymous telemetry data


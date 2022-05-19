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
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube](kubectl-testkube.md)	 - Testkube entrypoint for kubectl plugin
* [kubectl-testkube disable analytics](kubectl-testkube_disable_analytics.md)	 - disable collecting of anonymous analytics
* [kubectl-testkube disable oauth](kubectl-testkube_disable_oauth.md)	 - disable oauth authentication for direct api


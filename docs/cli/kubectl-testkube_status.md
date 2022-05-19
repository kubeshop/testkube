## kubectl-testkube status

Show status of feature or resource

```
kubectl-testkube status <feature|resource> [flags]
```

### Options

```
  -h, --help   help for status
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
* [kubectl-testkube status analytics](kubectl-testkube_status_analytics.md)	 - Get analytics status
* [kubectl-testkube status oauth](kubectl-testkube_status_oauth.md)	 - Get oauth status


## kubectl-testkube disable oauth

disable oauth authentication for direct api

```
kubectl-testkube disable oauth [flags]
```

### Options

```
  -h, --help   help for oauth
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

* [kubectl-testkube disable](kubectl-testkube_disable.md)	 - Disable feature


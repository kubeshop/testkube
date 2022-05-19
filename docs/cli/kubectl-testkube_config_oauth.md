## kubectl-testkube config oauth

Set oauth credentials for api uri in testkube client

```
kubectl-testkube config oauth <value> [flags]
```

### Options

```
      --auth-uri string        auth uri for authentication provider (github is a default provider) (default "https://github.com/login/oauth/authorize")
      --client-id string       client id for authentication provider
      --client-secret string   client secret for authentication provider
  -h, --help                   help for oauth
      --scope stringArray      scope for authentication provider
      --token-uri string       token uri for authentication provider (github is a default provider) (default "https://github.com/login/oauth/access_token")
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

* [kubectl-testkube config](kubectl-testkube_config.md)	 - Set feature configuration value


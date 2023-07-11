## testkube cloud login

Login to Testkube Cloud

```
testkube cloud login [flags]
```

### Options

```
      --cloud-root-domain string   defaults to testkube.io, usually don't need to be changed [required for cloud mode] (default "testkube.io")
      --env-id string              Testkube Cloud environment id
  -h, --help                       help for login
      --org-id string              Testkube Cloud organization id
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "https://demo.testkube.io/results/v1")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube cloud](testkube_cloud.md)	 - Testkube Cloud commands


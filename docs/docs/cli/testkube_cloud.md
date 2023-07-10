## testkube cloud

Testkube Cloud commands

```
testkube cloud [flags]
```

### Options

```
  -h, --help   help for cloud
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

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube cloud connect](testkube_cloud_connect.md)	 - Testkube Cloud connect 
* [testkube cloud disconnect](testkube_cloud_disconnect.md)	 - Switch back to Testkube OSS mode, based on active .kube/config file
* [testkube cloud init](testkube_cloud_init.md)	 - Install Helm chart registry in current kubectl context and update dependencies
* [testkube cloud login](testkube_cloud_login.md)	 - Login to Testkube Cloud


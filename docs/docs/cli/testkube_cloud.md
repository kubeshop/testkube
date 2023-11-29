## testkube cloud

[Deprecated] Testkube Cloud commands

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
      --insecure           insecure connection for direct client
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube cloud connect](testkube_cloud_connect.md)	 - [Deprecated] Testkube Cloud connect 
* [testkube cloud disconnect](testkube_cloud_disconnect.md)	 - [Deprecated] Switch back to Testkube OSS mode, based on active .kube/config file
* [testkube cloud init](testkube_cloud_init.md)	 - [Deprecated] Install Testkube Cloud Agent and connect to Testkube Cloud environment
* [testkube cloud login](testkube_cloud_login.md)	 - [Deprecated] Login to Testkube Pro


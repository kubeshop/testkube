## testkube pro

Testkube Pro commands

```
testkube pro [flags]
```

### Options

```
  -h, --help   help for pro
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube pro connect](testkube_pro_connect.md)	 - Testkube Pro connect 
* [testkube pro disconnect](testkube_pro_disconnect.md)	 - Switch back to Testkube Core OSS mode, based on active .kube/config file
* [testkube pro init](testkube_pro_init.md)	 - Install Testkube Pro Agent and connect to Testkube Pro environment
* [testkube pro login](testkube_pro_login.md)	 - Login to Testkube Pro


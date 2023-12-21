## testkube login

Login to Testkube Pro

```
testkube login [flags]
```

### Options

```
      --agent-prefix string   defaults to 'agent', usually don't need to be changed [required for custom cloud mode] (default "agent")
      --agent-token string    Testkube Cloud agent key [required for centralized mode]
      --agent-uri string      Testkube Cloud agent URI [required for centralized mode]
      --api-prefix string     defaults to 'api', usually don't need to be changed [required for custom cloud mode] (default "api")
      --env-id string         Testkube Cloud environment id [required for centralized mode]
  -h, --help                  help for login
      --master-insecure       should client connect in insecure mode (will use http instead of https)
      --org-id string         Testkube Cloud organization id [required for centralized mode]
      --root-domain string    defaults to testkube.io, usually don't need to be changed [required for custom cloud mode] (default "testkube.io")
      --ui-prefix string      defaults to 'ui', usually don't need to be changed [required for custom cloud mode] (default "ui")
```

### Options inherited from parent commands

```
  -a, --api-uri string     api uri, default value read from config if set (default "https://demo.testkube.io/results")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --insecure           insecure connection for direct client
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth
      --verbose            show additional debug messages
```

### SEE ALSO

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin


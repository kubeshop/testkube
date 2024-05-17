## testkube pro login

Login to Testkube Pro

```
testkube pro login [flags]
```

### Options

```
      --agent-prefix string   usually don't need to be changed [required for custom cloud mode] (default "agent")
      --agent-token string    Testkube Pro agent key [required for centralized mode]
      --agent-uri string      Testkube Pro agent URI [required for centralized mode]
      --api-prefix string     usually don't need to be changed [required for custom cloud mode] (default "api")
      --env-id string         Testkube Pro environment id [required for centralized mode]
      --feature-logs-v2       Logs v2 feature flag
  -h, --help                  help for login
      --logs-prefix string    usually don't need to be changed [required for custom cloud mode] (default "logs")
      --logs-uri string       Testkube Pro logs URI [required for centralized mode]
      --master-insecure       should client connect in insecure mode (will use http instead of https)
      --org-id string         Testkube Pro organization id [required for centralized mode]
      --root-domain string    usually don't need to be changed [required for custom cloud mode] (default "testkube.io")
      --ui-prefix string      usually don't need to be changed [required for custom cloud mode] (default "app")
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

* [testkube pro](testkube_pro.md)	 - Testkube Pro commands


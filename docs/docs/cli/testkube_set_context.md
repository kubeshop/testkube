## testkube set context

Set context data for Testkube Pro

```
testkube set context <value> [flags]
```

### Options

```
      --agent-prefix string         usually don't need to be changed [required for custom cloud mode] (default "agent")
      --agent-token string          Testkube Pro agent key [required for centralized mode]
      --agent-uri string            Testkube Pro agent URI [required for centralized mode]
      --agent-uri-override string   agnet uri override
  -k, --api-key string              API Key for Testkube Pro
      --api-prefix string           usually don't need to be changed [required for custom cloud mode] (default "api")
      --api-uri-override string     api uri override
      --env-id string               Testkube Pro environment id [required for centralized mode]
      --feature-logs-v2             Logs v2 feature flag
  -h, --help                        help for context
      --kubeconfig                  reset context mode for CLI to default kubeconfig based
      --logs-prefix string          usually don't need to be changed [required for custom cloud mode] (default "logs")
      --logs-uri string             Testkube Pro logs URI [required for centralized mode]
      --logs-uri-override string    logs service uri override
      --master-insecure             should client connect in insecure mode (will use http instead of https)
  -n, --namespace string            Testkube namespace to use for CLI commands
      --org-id string               Testkube Pro organization id [required for centralized mode]
      --root-domain string          usually don't need to be changed [required for custom cloud mode] (default "testkube.io")
      --ui-prefix string            usually don't need to be changed [required for custom cloud mode] (default "app")
      --ui-uri-override string      ui uri override
```

### Options inherited from parent commands

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
      --insecure                insecure connection for direct client
      --oauth-enabled           enable oauth
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube set](testkube_set.md)	 - Set resources


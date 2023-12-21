## testkube cloud init

[Deprecated] Install Testkube Cloud Agent and connect to Testkube Cloud environment

```
testkube cloud init [flags]
```

### Options

```
      --agent-prefix string   usually don't need to be changed [required for custom cloud mode] (default "agent")
      --agent-token string    Testkube Cloud agent key [required for centralized mode]
      --agent-uri string      Testkube Cloud agent URI [required for centralized mode]
      --api-prefix string     usually don't need to be changed [required for custom cloud mode] (default "api")
      --chart string          chart name (usually you don't need to change it) (default "kubeshop/testkube")
      --dry-run               dry run mode - only print commands that would be executed
      --env-id string         Testkube Cloud environment id [required for centralized mode]
  -h, --help                  help for init
      --master-insecure       should client connect in insecure mode (will use http instead of https)
      --multi-namespace       multi namespace mode
      --name string           installation name (usually you don't need to change it) (default "testkube")
      --namespace string      namespace where to install (default "testkube")
      --no-confirm            don't ask for confirmation - unatended installation mode
      --no-dashboard          don't install dashboard
      --no-minio              don't install MinIO
      --no-mongo              don't install MongoDB
      --no-operator           should operator be installed (for more instances in multi namespace mode it should be set to true)
      --org-id string         Testkube Cloud organization id [required for centralized mode]
      --root-domain string    usually don't need to be changed [required for custom cloud mode] (default "testkube.io")
      --ui-prefix string      usually don't need to be changed [required for custom cloud mode] (default "app")
      --values string         path to Helm values file
```

### Options inherited from parent commands

```
  -a, --api-uri string   api uri, default value read from config if set (default "https://demo.testkube.io/results")
  -c, --client string    client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --insecure         insecure connection for direct client
      --oauth-enabled    enable oauth
      --verbose          show additional debug messages
```

### SEE ALSO

* [testkube cloud](testkube_cloud.md)	 - [Deprecated] Testkube Cloud commands


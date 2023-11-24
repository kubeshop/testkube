## testkube pro connect

Testkube Pro connect

```
testkube pro connect [flags]
```

### Options

```
      --agent-token string         Testkube Pro agent key [required for pro mode]
      --chart string               chart name (usually you don't need to change it) (default "kubeshop/testkube")
      --pro-root-domain string   defaults to testkube.io, usually don't need to be changed [required for pro mode] (default "testkube.io")
      --dashboard-replicas int     Dashboard replicas
      --dry-run                    dry run mode - only print commands that would be executed
      --env-id string              Testkube Pro environment id [required for pro mode]
  -h, --help                       help for connect
      --minio-replicas int         MinIO replicas
      --mongo-replicas int         MongoDB replicas
      --name string                installation name (usually you don't need to change it) (default "testkube")
      --namespace string           namespace where to install (default "testkube")
      --no-confirm                 don't ask for confirmation - unatended installation mode
      --no-dashboard               don't install dashboard
      --no-minio                   don't install MinIO
      --no-mongo                   don't install MongoDB
      --org-id string              Testkube Pro organization id [required for pro mode]
      --values string              path to Helm values file
```

### Options inherited from parent commands

```
  -a, --api-uri string   api uri, default value read from config if set (default "https://demo.testkube.io/results/v1")
  -c, --client string    client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --oauth-enabled    enable oauth
      --verbose          show additional debug messages
```

### SEE ALSO

* [testkube pro](testkube_pro.md)	 - Testkube Pro commands


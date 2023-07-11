## testkube cloud init

Install Helm chart registry in current kubectl context and update dependencies

```
testkube cloud init [flags]
```

### Options

```
      --agent-token string         Testkube Cloud agent key
      --chart string               chart name (usually you don't need to change it) (default "kubeshop/testkube")
      --cloud-root-domain string   defaults to testkube.io, usually don't need to be changed [required for cloud mode] (default "testkube.io")
      --dry-run                    dry run mode - only print commands that would be executed
      --env-id string              Testkube Cloud environment id
  -h, --help                       help for init
      --name string                installation name (usually you don't need to change it) (default "testkube")
      --namespace string           namespace where to install (default "testkube")
      --no-confirm                 don't ask for confirmation - unatended installation mode
      --org-id string              Testkube Cloud organization id
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

* [testkube cloud](testkube_cloud.md)	 - Testkube Cloud commands


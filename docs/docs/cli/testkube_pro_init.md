## testkube pro init

Install Testkube Pro Agent and connect to Testkube Pro environment

```
testkube pro init [flags]
```

### Options

```
      --agent-token string         Testkube Pro agent key
      --chart string               chart name (usually you don't need to change it) (default "kubeshop/testkube")
      --pro-root-domain string   defaults to testkube.io, usually don't need to be changed [required for pro mode] (default "testkube.io")
      --dry-run                    dry run mode - only print commands that would be executed
      --env-id string              Testkube Pro environment id
  -h, --help                       help for init
      --multi-namespace            multi namespace mode
      --name string                installation name (usually you don't need to change it) (default "testkube")
      --namespace string           namespace where to install (default "testkube")
      --no-confirm                 don't ask for confirmation - unatended installation mode
      --no-operator                should operator be installed (for more instances in multi namespace mode it should be set to true)
      --org-id string              Testkube Pro organization id
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


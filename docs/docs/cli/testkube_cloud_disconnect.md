## testkube cloud disconnect

Switch back to Testkube OSS mode, based on active .kube/config file

```
testkube cloud disconnect [flags]
```

### Options

```
      --chart string             chart name (usually you don't need to change it) (default "kubeshop/testkube")
      --dashboard-replicas int   Dashboard replicas (default 1)
      --dry-run                  dry run mode - only print commands that would be executed
  -h, --help                     help for disconnect
      --minio-replicas int       MinIO replicas (default 1)
      --mongo-replicas int       MongoDB replicas (default 1)
      --name string              installation name (usually you don't need to change it) (default "testkube")
      --namespace string         namespace where to install (default "testkube")
      --no-confirm               don't ask for confirmation - unatended installation mode
      --no-dashboard             don't install dashboard
      --no-minio                 don't install MinIO
      --no-mongo                 don't install MongoDB
      --values string            path to Helm values file
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


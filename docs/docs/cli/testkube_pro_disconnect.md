## testkube pro disconnect

Switch back to Testkube Core OSS mode, based on active .kube/config file

```
testkube pro disconnect [flags]
```

### Options

```
      --chart string         chart name (usually you don't need to change it) (default "kubeshop/testkube")
      --dry-run              dry run mode - only print commands that would be executed
  -h, --help                 help for disconnect
      --minio-replicas int   MinIO replicas (default 1)
      --mongo-replicas int   MongoDB replicas (default 1)
      --name string          installation name (usually you don't need to change it) (default "testkube")
      --namespace string     namespace where to install (default "testkube")
      --no-confirm           don't ask for confirmation - unatended installation mode
      --no-minio             don't install MinIO
      --no-mongo             don't install MongoDB
      --values string        path to Helm values file
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

* [testkube pro](testkube_pro.md)	 - Testkube Pro commands


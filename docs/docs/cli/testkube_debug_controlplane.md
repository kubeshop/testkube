## testkube debug controlplane

Show Control Plane debug information

### Synopsis

Get all the necessary information to debug an issue in Testkube Control Plane you can fiter through comma separated list of items to show with additional flag `--show pods,services,ingresses,storageclasses,events,nats,connection,api,nats,mongo,dex,ui,worker`

```
testkube debug controlplane [flags]
```

### Options

```
  -h, --help            help for controlplane
  -s, --show []string   Comma-separated list of features to show, one of: pods,services,ingresses,storageclasses,events,nats,connection,api,nats,mongo,dex,ui,worker, defaults to all
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

* [testkube debug](testkube_debug.md)	 - Print debugging info


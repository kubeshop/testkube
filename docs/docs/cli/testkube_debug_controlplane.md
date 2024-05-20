## testkube debug controlplane

Show debug info

### Synopsis

Get all the necessary information to debug an issue in Testkube Control Plane

```
testkube debug controlplane [flags]
```

### Options

```
      --attach-agent-log        Attach agent log to the output keep in mind to configure valid agent first in the Testkube CLI
  -h, --help                    help for controlplane
      --labels stringToString   Labels to filter logs by (default [])
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

* [testkube debug](testkube_debug.md)	 - Print environment information for debugging


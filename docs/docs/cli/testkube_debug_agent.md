## testkube debug agent

Show Agent debug information

### Synopsis

Get all the necessary information to debug an issue in Testkube Agent you can fiter through comma separated list of items to show with additional flag `--show pods,services,ingresses,events,nats,connection,roundtrip`

```
testkube debug agent [flags]
```

### Options

```
  -h, --help            help for agent
  -s, --show []string   Comma-separated list of features to show, one of: pods,services,ingresses,events,nats,connection,roundtrip, defaults to all
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


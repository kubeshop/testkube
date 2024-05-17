## testkube disable

Disable feature

```
testkube disable <feature> [flags]
```

### Options

```
  -h, --help   help for disable
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

* [testkube](testkube.md)	 - Testkube entrypoint for kubectl plugin
* [testkube disable oauth](testkube_disable_oauth.md)	 - disable oauth authentication for direct api
* [testkube disable telemetry](testkube_disable_telemetry.md)	 - disable collecting of anonymous telemetry data


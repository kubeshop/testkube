## testkube init demo

Install Testkube On-Prem demo in your current context

```
testkube init demo [flags]
```

### Options

```
      --dry-run            Dry run
      --export             Export the values.yaml
  -h, --help               help for demo
  -l, --license string     License key
  -n, --namespace string   Namespace to install Testkube On-Prem demo
  -y, --no-confirm         Skip confirmation
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

* [testkube init](testkube_init.md)	 - Init Testkube profiles(standalone-agent|demo|agent)


## testkube init

Init Testkube profiles(standalone-agent|demo|agent)

### Synopsis

Init installs the Testkube in your cluster as follows:
	standalone-agent -> Testkube OSS
	demo -> Testkube On-Prem demo
	agent -> Testkube Agent

```
testkube init <profile> [flags]
```

### Options

```
      --export   Export the values.yaml
  -h, --help     help for init
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
* [testkube init demo](testkube_init_demo.md)	 - Install Testkube On-Prem demo in your current context
* [testkube init init](testkube_init_init.md)	 - Install Testkube Pro Agent and connect to Testkube Pro environment
* [testkube init standalone-agent](testkube_init_standalone-agent.md)	 - Install Testkube OSS in your current context


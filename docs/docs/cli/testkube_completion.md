## testkube completion

Generate the autocompletion script for the specified shell

### Synopsis

Generate the autocompletion script for testkube for the specified shell.
See each sub-command's help for details on how to use the generated script.


### Options

```
  -h, --help   help for completion
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
* [testkube completion bash](testkube_completion_bash.md)	 - Generate the autocompletion script for bash
* [testkube completion fish](testkube_completion_fish.md)	 - Generate the autocompletion script for fish
* [testkube completion powershell](testkube_completion_powershell.md)	 - Generate the autocompletion script for powershell
* [testkube completion zsh](testkube_completion_zsh.md)	 - Generate the autocompletion script for zsh


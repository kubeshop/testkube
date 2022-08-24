## kubectl-testkube completion zsh

Generate the autocompletion script for zsh

### Synopsis

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(kubectl-testkube completion zsh); compdef _kubectl-testkube kubectl-testkube

To load completions for every new session, execute once:

#### Linux:

	kubectl-testkube completion zsh > "${fpath[1]}/_kubectl-testkube"

#### macOS:

	kubectl-testkube completion zsh > $(brew --prefix)/share/zsh/site-functions/_kubectl-testkube

You will need to start a new shell for this setup to take effect.


```
kubectl-testkube completion zsh [flags]
```

### Options

```
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data (default true)
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube completion](kubectl-testkube_completion.md)	 - Generate the autocompletion script for the specified shell


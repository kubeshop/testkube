## kubectl-testkube completion zsh

generate the autocompletion script for zsh

### Synopsis


Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:
# Linux:
$ kubectl-testkube completion zsh > "${fpath[1]}/_kubectl-testkube"
# macOS:
$ kubectl-testkube completion zsh > /usr/local/share/zsh/site-functions/_kubectl-testkube

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
      --analytics-enabled   enable analytics
  -w, --api-uri string      api uri, default value read from config if set (default "http://testdash.testkube.io/api")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube completion](kubectl-testkube_completion.md)	 - generate the autocompletion script for the specified shell


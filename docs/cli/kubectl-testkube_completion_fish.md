## kubectl-testkube completion fish

generate the autocompletion script for fish

### Synopsis


Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
$ kubectl-testkube completion fish | source

To load completions for every new session, execute once:
$ kubectl-testkube completion fish > ~/.config/fish/completions/kubectl-testkube.fish

You will need to start a new shell for this setup to take effect.


```
kubectl-testkube completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics (default true)
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube completion](kubectl-testkube_completion.md)	 - generate the autocompletion script for the specified shell


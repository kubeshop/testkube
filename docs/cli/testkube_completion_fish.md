## testkube completion fish

generate the autocompletion script for fish

### Synopsis


Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
$ testkube completion fish | source

To load completions for every new session, execute once:
$ testkube completion fish > ~/.config/fish/completions/testkube.fish

You will need to start a new shell for this setup to take effect.


```
testkube completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --analytics-enabled   Enable analytics (default true)
  -c, --client string       Client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace (default "testkube")
  -v, --verbose             Show additional debug messages
```

### SEE ALSO

* [testkube completion](testkube_completion.md)	 - generate the autocompletion script for the specified shell


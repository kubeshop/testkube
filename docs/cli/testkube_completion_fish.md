## testkube completion fish

generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

```sh
testkube completion fish | source
```

To load completions for every new session, execute once:

```sh
testkube completion fish > ~/.config/fish/completions/testkube.fish
```

You will need to start a new shell for this setup to take effect.

```sh
testkube completion fish [flags]
```

### Options

```sh
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```sh
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string   kubernetes namespace (default "testkube")
  -v, --verbose            should I show additional debug messages
```

### SEE ALSO

* [testkube completion](testkube_completion.md)  - generate the autocompletion script for the specified shell

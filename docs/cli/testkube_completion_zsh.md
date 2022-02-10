## testkube completion zsh

generate the autocompletion test for zsh

### Synopsis

Generate the autocompletion test for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

```sh
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

To load completions for every new session, execute once:

### Linux

```sh
testkube completion zsh > "${fpath[1]}/_testkube"
```

### macOS

```sh
testkube completion zsh > /usr/local/share/zsh/site-functions/_testkube
```

You will need to start a new shell for this setup to take effect.

```sh
testkube completion zsh [flags]
```

### Options

```sh
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```sh
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string   kubernetes namespace (default "testkube")
  -v, --verbose            should I show additional debug messages
```

### SEE ALSO

* [testkube completion](testkube_completion.md)  - generate the autocompletion test for the specified shell

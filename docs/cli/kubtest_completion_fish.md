## kubtest completion fish

generate the autocompletion script for fish

### Synopsis


Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
$ kubtest completion fish | source

To load completions for every new session, execute once:
$ kubtest completion fish > ~/.config/fish/completions/kubtest.fish

You will need to start a new shell for this setup to take effect.


```
kubtest completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### SEE ALSO

* [kubtest completion](kubtest_completion.md)	 - generate the autocompletion script for the specified shell


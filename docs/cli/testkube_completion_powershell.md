## testkube completion powershell

generate the autocompletion test for powershell

### Synopsis

Generate the autocompletion test for powershell.

To load completions in your current shell session:

```powershell
testkube completion powershell | Out-String | Invoke-Expression
```

To load completions for every new session, add the output of the above command
to your powershell profile.

```powershell
testkube completion powershell [flags]
```

### Options

```powershell
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```powershell
  -c, --client string      Client used for connecting to testkube API one of proxy|direct (default "proxy")
  -s, --namespace string   kubernetes namespace (default "testkube")
  -v, --verbose            should I show additional debug messages
```

### SEE ALSO

* [testkube completion](testkube_completion.md)  - generate the autocompletion test for the specified shell

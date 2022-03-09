# Testkube Completion Fish

## **Synopsis**

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
```
$ testkube completion fish | source
```

To load completions for every new session and execute once:
```
$ testkube completion fish > ~/.config/fish/completions/testkube.fish
```

You will need to start a new shell for this setup to take effect.


```
testkube completion fish [flags]
```

## **Options**

```
  -h, --help              Help for fish.
      --no-descriptions   Disable completion descriptions.
```

## **Options Inherited from Parent Commands**

```
      --analytics-enabled   Enable analytics (default "true").
  -c, --client string       Client used for connecting to testkube API one of proxy|direct (default "proxy").
  -s, --namespace string    Kubernetes namespace (default "testkube").
  -v, --verbose             Show additional debug messages.
```

## **SEE ALSO**

* [Testkube Completion](testkube_completion.md)	 - Generate the autocompletion script for the specified shell.


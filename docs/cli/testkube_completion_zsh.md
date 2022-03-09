# Testkube Completion Zsh

## **Synopsis**


Generate the autocompletion script for the Zsh shell.

If shell completion is not already enabled in your environment, you will need
to enable it.  You can execute the following once:
```
$ echo "autoload -U compinit; compinit" >> ~/.zshrc
```
To load completions for every new session and execute once:
```
# Linux:
$ testkube completion zsh > "${fpath[1]}/_testkube"
# macOS:
$ testkube completion zsh > /usr/local/share/zsh/site-functions/_testkube
```

You will need to start a new shell for this setup to take effect:


```
testkube completion zsh [flags]
```

## **Options**

```
  -h, --help              Help for Zsh.
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


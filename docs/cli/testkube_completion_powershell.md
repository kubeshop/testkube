# Testkube Completion Powershell

## **Synopsis**

Generate the autocompletion script for Powershell.

To load completions in your current shell session:
```
PS C:\> testkube completion powershell | Out-String | Invoke-Expression
```

To load completions for every new session, add the output of the above command to your Powershell profile.


```
testkube completion powershell [flags]
```

## **Options**

```
  -h, --help              Help for Powershell.
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


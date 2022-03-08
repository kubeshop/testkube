# Testkube Completion Bash

## **Synopsis**


Generate the autocompletion script for the Bash shell.

This script depends on the 'bash-completion' package.
If not previously installed, install it via your OS's package manager.

To load completions in your current shell session:
```
$ source <(testkube completion bash)
```

To load completions for every new session and 
execute once:
```
Linux:
  $ testkube completion bash > /etc/bash_completion.d/testkube
MacOS:
  $ testkube completion bash > /usr/local/etc/bash_completion.d/testkube
```

You will need to start a new shell for this setup to take effect:
  

```
testkube completion bash
```

## **Options**

```
  -h, --help              Help for Bash.
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


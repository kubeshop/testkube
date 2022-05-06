## kubectl-testkube completion bash

generate the autocompletion script for bash

### Synopsis


Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:
$ source <(kubectl-testkube completion bash)

To load completions for every new session, execute once:
Linux:
  $ kubectl-testkube completion bash > /etc/bash_completion.d/kubectl-testkube
MacOS:
  $ kubectl-testkube completion bash > /usr/local/etc/bash_completion.d/kubectl-testkube

You will need to start a new shell for this setup to take effect.
  

```
kubectl-testkube completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --analytics-enabled   enable analytics
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -s, --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
  -v, --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube completion](kubectl-testkube_completion.md)	 - generate the autocompletion script for the specified shell


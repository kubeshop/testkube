## kubectl-testkube completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(kubectl-testkube completion bash)

To load completions for every new session, execute once:

#### Linux:

	kubectl-testkube completion bash > /etc/bash_completion.d/kubectl-testkube

#### macOS:

	kubectl-testkube completion bash > $(brew --prefix)/etc/bash_completion.d/kubectl-testkube

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
  -a, --api-uri string      api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string       client used for connecting to Testkube API one of proxy|direct (default "proxy")
      --namespace string    Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled       enable oauth
      --telemetry-enabled   enable collection of anonumous telemetry data (default true)
      --verbose             show additional debug messages
```

### SEE ALSO

* [kubectl-testkube completion](kubectl-testkube_completion.md)	 - Generate the autocompletion script for the specified shell


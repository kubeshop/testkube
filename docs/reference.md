```

██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                               /kjuːb tɛst/ by Kubeshop


```
# Commands reference for `kubectl kubetest` plugin

Commands tree
```
completion
	bash
	fish
	powershell
	zsh
doc
help
install
scripts
	abort
	create
	execution
	executions
	list
	start
version
```


## `completion` command


```

Generate the autocompletion script for  for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
   completion [command]

Available Commands:
  bash        generate the autocompletion script for bash
  fish        generate the autocompletion script for fish
  powershell  generate the autocompletion script for powershell
  zsh         generate the autocompletion script for zsh

Use " completion [command] --help" for more information about a command.
```


### `bash` command


```

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:
$ source <( completion bash)

To load completions for every new session, execute once:
Linux:
  $  completion bash > /etc/bash_completion.d/
MacOS:
  $  completion bash > /usr/local/etc/bash_completion.d/

You will need to start a new shell for this setup to take effect.

Usage:
   completion bash

Flags:
      --no-descriptions   disable completion descriptions
```


### `fish` command


```

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
$  completion fish | source

To load completions for every new session, execute once:
$  completion fish > ~/.config/fish/completions/.fish

You will need to start a new shell for this setup to take effect.

Usage:
   completion fish [flags]

Flags:
      --no-descriptions   disable completion descriptions
```


### `powershell` command


```

Generate the autocompletion script for powershell.

To load completions in your current shell session:
PS C:\>  completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

Usage:
   completion powershell [flags]

Flags:
      --no-descriptions   disable completion descriptions
```


### `zsh` command


```

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for every new session, execute once:
# Linux:
$  completion zsh > "${fpath[1]}/_"
# macOS:
$  completion zsh > /usr/local/share/zsh/site-functions/_

You will need to start a new shell for this setup to take effect.

Usage:
   completion zsh [flags]

Flags:
      --no-descriptions   disable completion descriptions
```


## `doc` command


```
Generate docs for kubectl kubtest

Usage:
   doc [flags]

Flags:
  -h, --help   help for doc
```


## `help` command


```
Help provides help for any command in the application.
Simply type  help [path to command] for full details.

Usage:
   help [command]
```


## `install` command


```
Install can be configured with use of particular

Usage:
   install [flags]

Flags:
      --chart string       chart name (default "kubeshop/kubtest")
      --name string        installation name (default "kubtest")
      --namespace string   namespace where to install (default "default")
```


## `scripts` command


```
All available scripts and scripts executions commands

Usage:
   scripts [flags]
   scripts [command]

Available Commands:
  abort       (NOT IMPLEMENTED) Aborts execution of the script
  create      Create new script
  execution   Gets script execution details
  executions  List scripts executions
  list        Get all available scripts
  start       Starts new script

Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages

Use " scripts [command] --help" for more information about a command.
```


### `abort` command


```
(NOT IMPLEMENTED) Aborts execution of the script

Usage:
   scripts abort [flags]

Global Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```


### `create` command


```
Create new Script Custom Resource,

Usage:
   scripts create [flags]

Flags:
  -f, --file string        script file - will be read from stdin if not specified
  -n, --name string        unique script name - mandatory
  -s, --namespace string   kubernetes namespace where script will be created (default "default")
  -t, --type string        script type (defaults to postman-collection) (default "postman/collection")

Global Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```


### `execution` command


```
Gets script execution details, you can change output format

Usage:
   scripts execution [flags]

Global Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```


### `executions` command


```
Getting list of execution for given script name or recent executions if there is no script name passed

Usage:
   scripts executions [flags]

Global Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```


### `list` command


```
Getting all available scritps from given namespace - if no namespace given "default" namespace is used

Usage:
   scripts list [flags]

Flags:
  -s, --namespace string   kubernetes namespace from where scripts should be listed (default "default")

Global Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```


### `start` command


```
Starts new script based on Script Custom Resource name, returns results to console

Usage:
   scripts start [flags]

Flags:
  -n, --name string            execution name, if empty will be autogenerated
  -s, --namespace string       scripts kubernetes namespace (default "default")
  -p, --param stringToString   execution envs passed to executor (default [])

Global Flags:
  -c, --client string        Client used for connecting to kubtest API one of proxy|direct (default "proxy")
      --go-template string   in case of choosing output==go pass golang template (default "{{ . | printf \"%+v\"  }}")
  -o, --output string        output typoe one of raw|json|go  (default "raw")
  -v, --verbose              should I show additional debug messages
```


## `version` command


```
Shows version and build info

Usage:
   version
```



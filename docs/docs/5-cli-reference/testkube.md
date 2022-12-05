## kubectl-testkube

Testkube entrypoint for kubectl plugin

```
kubectl-testkube [flags]
```

### Options

```
  -a, --api-uri string     api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string      client used for connecting to Testkube API one of proxy|direct (default "proxy")
  -h, --help               help for kubectl-testkube
      --namespace string   Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled      enable oauth (default true)
      --verbose            show additional debug messages
```

### SEE ALSO

* [kubectl-testkube abort](kubectl-testkube_abort.md)	 - Abort tests or test suites
* [kubectl-testkube completion](kubectl-testkube_completion.md)	 - Generate the autocompletion script for the specified shell
* [kubectl-testkube config](kubectl-testkube_config.md)	 - Set feature configuration value
* [kubectl-testkube create](kubectl-testkube_create.md)	 - Create resource
* [kubectl-testkube create-ticket](kubectl-testkube_create-ticket.md)	 - Create bug ticket
* [kubectl-testkube dashboard](kubectl-testkube_dashboard.md)	 - Open testkube dashboard
* [kubectl-testkube debug](kubectl-testkube_debug.md)	 - Print environment information for debugging
* [kubectl-testkube delete](kubectl-testkube_delete.md)	 - Delete resources
* [kubectl-testkube disable](kubectl-testkube_disable.md)	 - Disable feature
* [kubectl-testkube download](kubectl-testkube_download.md)	 - Artifacts management commands
* [kubectl-testkube enable](kubectl-testkube_enable.md)	 - Enable feature
* [kubectl-testkube generate](kubectl-testkube_generate.md)	 - Generate resources commands
* [kubectl-testkube get](kubectl-testkube_get.md)	 - Get resources
* [kubectl-testkube init](kubectl-testkube_init.md)	 - Install Helm chart registry in current kubectl context and update dependencies
* [kubectl-testkube migrate](kubectl-testkube_migrate.md)	 - manual migrate command
* [kubectl-testkube purge](kubectl-testkube_purge.md)	 - Uninstall Helm chart registry from current kubectl context
* [kubectl-testkube run](kubectl-testkube_run.md)	 - Runs tests or test suites
* [kubectl-testkube status](kubectl-testkube_status.md)	 - Show status of feature or resource
* [kubectl-testkube update](kubectl-testkube_update.md)	 - Update resource
* [kubectl-testkube upgrade](kubectl-testkube_upgrade.md)	 - Upgrade Helm chart, install dependencies and run migrations
* [kubectl-testkube version](kubectl-testkube_version.md)	 - Shows version and build info
* [kubectl-testkube watch](kubectl-testkube_watch.md)	 - Watch tests or test suites


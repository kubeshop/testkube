## testkube

Testkube entrypoint for kubectl plugin

```
testkube [flags]
```

### Options

```
  -a, --api-uri string          api uri, default value read from config if set (default "http://localhost:8088")
  -c, --client string           client used for connecting to Testkube API one of proxy|direct|cluster (default "proxy")
      --header stringToString   headers for direct client key value pair: --header name=value (default [])
  -h, --help                    help for testkube
      --insecure                insecure connection for direct client
      --namespace string        Kubernetes namespace, default value read from config if set (default "testkube")
      --oauth-enabled           enable oauth
      --verbose                 show additional debug messages
```

### SEE ALSO

* [testkube abort](testkube_abort.md)	 - Abort tests or test suites
* [testkube agent](testkube_agent.md)	 - Testkube Pro Agent related commands
* [testkube completion](testkube_completion.md)	 - Generate the autocompletion script for the specified shell
* [testkube config](testkube_config.md)	 - Set feature configuration value
* [testkube create](testkube_create.md)	 - Create resource
* [testkube create-ticket](testkube_create-ticket.md)	 - Create bug ticket
* [testkube dashboard](testkube_dashboard.md)	 - Open Testkube dashboard
* [testkube debug](testkube_debug.md)	 - Print environment information for debugging
* [testkube delete](testkube_delete.md)	 - Delete resources
* [testkube disable](testkube_disable.md)	 - Disable feature
* [testkube download](testkube_download.md)	 - Artifacts management commands
* [testkube enable](testkube_enable.md)	 - Enable feature
* [testkube generate](testkube_generate.md)	 - Generate resources commands
* [testkube get](testkube_get.md)	 - Get resources
* [testkube init](testkube_init.md)	 - Init Testkube profiles(standalone-agent|demo|agent)
* [testkube login](testkube_login.md)	 - Login to Testkube Pro
* [testkube migrate](testkube_migrate.md)	 - manual migrate command
* [testkube pro](testkube_pro.md)	 - Testkube Pro commands
* [testkube purge](testkube_purge.md)	 - Uninstall Testkube from your current kubectl context
* [testkube run](testkube_run.md)	 - Runs tests, test suites or test workflows
* [testkube set](testkube_set.md)	 - Set resources
* [testkube status](testkube_status.md)	 - Show status of feature or resource
* [testkube update](testkube_update.md)	 - Update resource
* [testkube upgrade](testkube_upgrade.md)	 - Upgrade Helm chart, install dependencies and run migrations
* [testkube version](testkube_version.md)	 - Shows version and build info
* [testkube watch](testkube_watch.md)	 - Watch tests or test suites


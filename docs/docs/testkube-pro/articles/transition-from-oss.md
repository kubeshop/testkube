# Migrating from Testkube Core Open Source

If you have started using Testkube using the Open Source installation, you can migrate this instance to be managed using Testkube Pro.

To connect your Testkube Core Open Source instance, you will need to modify your Testkube installation to be in Pro Agent mode. Testkube Pro Agent is the Testkube engine for controlling your Testkube instance using the managed solution. It sends data to Testkube's Pro Servers.

::: note

Currently, we do not support uploading existing test logs and artifacts from your Testkube Core Open Source instance. This is planned for coming releases.

::: 

## Instructions

1. Run the following command which will walk you through the migration process:

```sh
testkube pro connect
```

2. [Set your CLI Context to talk to Testkube Pro](./managing-cli-context.md).
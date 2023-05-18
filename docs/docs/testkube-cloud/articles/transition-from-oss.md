# Migrating from Testkube Open Source

If you have started using Testkube using the Open Source installation, you can migrate this instance to be managed using Testkube Cloud. 

To connect your Testkube Open Source instance, you will need to modify your Testkube installation to be in Cloud Agent mode. Testkube Cloud Agent is the Testkube engine for controlling your Testkube instance using the managed solution. It sends data to Testkube's Cloud Servers.

::: note

Currently, we do not support uploading existing test logs and artifacts from your Testkube Open Source instance. This is planned for coming releases.

::: 

## Instructions

1. Run the following command which will walk you through the migration process:

```sh
testkube cloud connect
```

2. [Set your CLI Context to talk to Testkube Cloud](./managing-cli-context.md).
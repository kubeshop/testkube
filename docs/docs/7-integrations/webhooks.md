---
sidebar_position: 4
sidebar_label: Implementing Webhooks
---
# Implementing Webhooks

Webhooks allow you to build or set up integrations and send HTTP POST payloads (your Testkube Execution and its current state) whenever an event is triggered. In this case, when your Tests are started or finished.

To set them up when using Testkube, you'll need to create your webhook as shown in the following format example and `kubectl apply` it:

```yml
apiVersion: executor.testkube.io/v1
kind: Webhook
metadata:
  name: example-webhook
  namespace: testkube
spec:
  uri: http://localhost:8080/events
  events:
  - start-test
  - end-test
  - end-test-success
  - end-test-failed
```

Here you'll be able to pass events depending on which webhooks you want to be triggered. Testkube will pass `Event` which can have `type` and `testExecution` or `testsuiteExecution` fields.

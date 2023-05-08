# Webhooks

[Webhooks](https://docs.github.com/en/webhooks-and-events/webhooks/about-webhooks) allow you to build or set up integrations and send HTTP POST payloads (your Testkube Execution and its current state) whenever an event is triggered. In this case, when your Tests start or finish.

To set them up when using Testkube, you'll need to create your webhook as shown in the following format example:

```yaml title="webhook.yaml"
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

And then apply with: 

```sh 
kubectl apply -f webhook.yaml
```

Here you'll be able to pass events depending on which webhooks you want to be triggered. Testkube will pass `Event` which can have `type` and `testExecution` or `testsuiteExecution` fields.

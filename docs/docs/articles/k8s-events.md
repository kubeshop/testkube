# Emitting Kubernetes Events

Testkube can emit Kubernetes Events to your cluster. This can be used to integrate with any CD tool that supports the Kubernetes Events.

## Step 1 - Enable/Disable Kubernetes Events

To enable/disable Kubernets Events, you need to set the following Helm
values (it's enabled by default):

```sh
helm upgrade \
  --install \
  --create-namespace \
  --namespace testkube \
  testkube \
  kubeshop/testkube \
  --set testkube-api.enableK8sEvents=true
```

## Step 2 - Test Emmiting Kubernetes Events

To test emitting Kubernetes Events, create a sample test with Testkube and run it.

```sh
cat << EOF | > curl-test.json
{
  "command": [
    "curl",
    "https://example.com"
  ],
  "expected_status": "200"
}
EOF

testkube create test --name test-k8sevents --type curl/test -f curl-test.json

testkube run test test-k8sevents

kubectl describe events -n testkube
```

Check your kubernetes event list. An event like the following should have been emmitted: 

```yaml
Name:             testkube-event-0b54d591-f496-4980-bd69-ab6d093617e2
Namespace:        testkube
Labels:           executor=curl-executor
                  test-type=curl-test
Annotations:      <none>
Action:           started
API Version:      v1
Event Time:       2024-05-14T10:12:11.284683Z
First Timestamp:  2024-05-14T10:12:11Z
Involved Object:
  API Version:   tests.testkube.io/v3
  Kind:          Test
  Name:          test-k8sevents
  Namespace:     testkube
Kind:            Event
Last Timestamp:  2024-05-14T10:12:11Z
Message:         executionId=664338fb75921694f0a3222e
Metadata:
  Creation Timestamp:  2024-05-14T10:12:11Z
  Resource Version:    76609
  UID:                 9e5fac33-b958-42d6-bebc-45cd024a9ecc
Reason:                start-test
Reporting Component:   testkkube.io/services
Reporting Instance:    testkkube.io/services/testkube-api-server
Source:
Type:    Normal
Events:  <none>
```

## Reference

For more information about Kubernetes Events, please visit the [Kubernetes Events](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/) website.

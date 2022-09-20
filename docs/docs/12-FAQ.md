# FAQ
Frequently asked questions regarding your Testkube installation.
### Why is the UI not working or does not return results?

- Make sure the API endpoint is configured:

![img.gif](img/check-dashboard-api-endpoint.gif)

- Make sure the endpoint is providing data, e.g. accessing the executors path:

```sh
curl <endpoint>/v1/executors 
```

- If no data is provided, make sure that all the Testkube components are running properly:

```sh
kubectl get pods -n testkube
NAME                                                        READY   STATUS    RESTARTS   AGE
pod/testkube-api-server-8445fd7b9f-jq5rh                    1/1     Running   0          10d
pod/testkube-dashboard-99f4c6cf5-x4dkz                      1/1     Running   0          12d
pod/testkube-minio-testkube-76786f8f64-9nl4c                1/1     Running   1          24d
pod/testkube-mongodb-74587998bb-8pzl2                       1/1     Running   0          12d
pod/testkube-operator-controller-manager-77ffbb8fdc-rxhvx   2/2     Running   0          5d23h
```

- If these options do not solve the problem, please contact us on [Discord](https://discord.com/invite/6zupCZFQbe).
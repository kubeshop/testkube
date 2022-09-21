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

### Access the Service Under Test(SUT) Using Testkube

- Services inside the same Kubernetes cluster can be accessed using the address \<service-name\>.\<service-namespace\>.svc.cluster.local:\<port-number\>. If there are network restrictions configured, Testkube will need permissions to access the SUT over the local network of the cluster.
- If Testkube and the SUT are not in the same cluster, SUT will have to be exposed to Testkube using an Ingress or a Load Balancer.

If this does not solve the issue that you encountered or you have other questions or comments, please contact us on [Discord](https://discord.com/invite/6zupCZFQbe).
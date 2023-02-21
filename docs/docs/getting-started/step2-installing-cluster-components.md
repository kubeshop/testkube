# Step 2 - Install Testkube Cluster components using Testkube's CLI

The Testkube CLI provides a command to easily deploy the Testkube server components to your cluster.
Run:

```bash
testkube init
```

note: you must have your KUBECONFIG pointing to the desired location of the installation.

The above command will install the following components in your Kubernetes cluster:

1. Testkube API
2. `testkube` namespace
3. CRDs for Tests, TestSuites, Executors
4. MongoDB
5. Minio - default (can be disabled with `--no-minio`)
6. Dashboard - default (can be disabled with `--no-dashboard` flag)

Confirm that Testkube is running:

```bash
kubectl get all -n testkube
```

By default Testkube is installed in the `testkube` namespace.
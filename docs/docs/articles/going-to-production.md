# Overview

This section provides guidance to get Testkube to production, in particular to enable access to Testkube without requiring k8s privileges to the Testkube k8s namespace and workloads:
1. Overall [Testkube deployment on AWS](./deploying-in-aws.md).
2. Add [OAuth authentication to the Testkube api-server used by the CLI](./oauth-cli.md) as an alternative to default `proxy` mode leveraging [kube apiserver proxy](https://kubernetes.io/docs/concepts/cluster-administration/proxies/).   

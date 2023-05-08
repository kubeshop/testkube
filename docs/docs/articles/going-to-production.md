# Overview

This sections provides guidance to get Testkube to production, in particular to enable access to Testkube without requiring k8s privileges to the Testkube k8s namespace and workloads:
1. Exposing Testkube Dashboard [externally with ingresses](./exposing-testkube.md).
2. Overall [Testkube deployment on AWS](./deploying-in-aws.md).
3. Add [OAuth authentication to the Testkube dashboard](./oauth-dashboard.md). 
4. Add [OAuth authentication to the Testkube api-server used by the CLI](./oauth-cli.md) as an alternative to default `proxy` mode leveraging [kube apiserver proxy](https://kubernetes.io/docs/concepts/cluster-administration/proxies/).   

# Overview

This sections provides guidance to get testkube to production, in particular enable access to testkube without requiring k8s privileges to testkube k8s namespace and workloads:
1. Exposing Testkube Dashboard [externally with ingresses](exposing-testkube/overview.md) 
2. Overall [testkube deployment on AWS](aws.md)
3. Add [OAuth authentication to the Testkube dashboard](authentication/oauth-ui.md). 
4. Add [OAuth authentication to the Testkube api-server used by the CLI](authentication/oauth-cli.md) as an alternative to default `proxy` mode leveraging [kube apiserver proxy](https://kubernetes.io/docs/concepts/cluster-administration/proxies/).   

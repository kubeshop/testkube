# Overview

In this section you will:

1. [Install the Testkube CLI](./step1-installing-cli).
2. [Install the Testkube Agent](./step2-installing-cluster-components.md).
3. [Creating your first Test](./step3-creating-first-test.md).

You can also see the full installation video from our product experts: [Testkube Installation Video](https://www.youtube.com/watch?v=bjQboi3Etys):

<iframe width="100%" height="315" src="https://www.youtube.com/embed/ynzEkOUhxKk" title="YouTube Tutorial: Getting started with Testing in Kubernetes Using Testkube" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

In summary, you now have Testkube setup in your Kubernetes cluster, ready to discover the Testkube features:
- The Testkube CLI allows you to locally port forward to the kube dashboard deployment.
- The Testkube CLI allows you to interact with the Testkube server components through k8s CRs or REST API calls made through the [kube apiserver proxy](https://kubernetes.io/docs/concepts/cluster-administration/proxies/).

As usage of Testkube grows within your team, you may choose to:
* Leverage [managed Testkube cloud](../testkube-cloud/articles/intro.md).
* [Move to production](./going-to-production.md) with your own Testkube installation.
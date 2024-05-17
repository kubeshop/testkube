# Installation FAQ

## Can Testkube be deployed on OpenShift?

Yes, OpenShift is supported. You might need to tweak the configuration a bit to fit the security requirements. Feel free to contact us if it does not work out, and we’ll gladly hot fix it.

## Can Testkube OSS be migrated to join a control plane?

Yes, you can choose to get started with just the standalone agent. Once you are ready to use a control plane, you can join it with a control plane by following [this tutorial][migrate-oss].

## Do I have to have my own Kubernetes Cluster to evaluate Testkube

No, for evaluation you can use our SaaS offering at app.testkube.io which includes a Demo environment for you to play with. Read more about evaluating 
Testkube without having a Kubernetes cluster at [Quickstart without Kubernetes](quickstart-no-k8s.mdx)

## Do I have to have my own Kubernetes cluster to run Testkube in production

Yes, the Testkube Agent always runs in your own cluster(s)/infrastructure for managing and executing your tests. 
The Control Plane containing the Dashboard can be hosted either by us or by you. Read more about the Testkube architecture at
[Reference Architectures](reference-architectures.md)

## Can I run Testkube in an air-gapped environment

Yes, you can download and install Testkube in your airgapped environment as long as it has access to dockerhub (for example via artifactory) to retrieve the Testkube images. 
If that doesn't work for you please [get in touch](https://testkube.io/contact), and we will help you install Testkube as required.

## Can I use Testkube to test applications or services that are not running in Kubernetes

Yes, you can use Testkube to test any applications or components as long as the cluster the Testkube agent is running in has network access to the applications
or components to be tested.

[migrate-oss]: /testkube-pro-on-prem/articles/migrating-from-oss-to-pro

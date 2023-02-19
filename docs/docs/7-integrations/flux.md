# Flux + Testkube

## Challenges to GitOps Cloud Native Testing with Flux

One of the major trends in contemporary cloud native application development is the adoption of GitOps; managing the state of your Kubernetes cluster(s) in Git - with all the bells and whistles provided by modern Git platforms like GitHub and GitLab in regard to workflows, auditing, security, tooling, etc. Tools like ArgoCD or Flux are used to do the heavy lifting of keeping your Kubernetes cluster in sync with your Git repository; as soon as difference is detected between Git and your cluster it is deployed to ensure that your repository is the source-of-truth for your runtime environment.

We at Kubeshop are working hard to provide you with the first GitOps-friendly Cloud-native test orchestration/execution framework - Testkube - to ensure that your QA efforts align with this new approach to application configuration and cluster configuration management. Combined with the GitOps approach described above, Testkube will include your test artifacts and application configuration in the state of your cluster and make git the source of truth for these test artifacts. And it’s Open-Source too. For more on Testkube, check out the introduction blog, [Hello Testkube](https://testkube.io/blog/hello-testkube-power-to-testers-on-k8s).

## Benefits of the GitOps Approach

- Since your tests are included in the state of your cluster you are always able to validate that your application components/services work as required.
- Since tests are executed from inside your cluster there is no need to expose services under test externally purely for the purpose of being able to test them.
- Tests in your cluster are always in sync with the external tooling used for authoring
- Test execution is not strictly tied to CI but can also be triggered manually for ad-hoc validations or via internal triggers (Kubernetes events)
- You can leverage all your existing test automation assets from Postman, or Cypress (even for end-to-end testing), or … through executor plugins.

Conceptually, this can be illustrated as follows:

![GitOps CLoud Testing](../img/flux.png)

## GitOps Tutorial

The following is a step-by-step walkthrough to get this in place for the automated application deployment and execution of Postman collections in a local Kind cluster to test.

Let’s start with setting things up for our GitOps-powered testing machine!

### **Installations for GitOps Testing**
**1.** [Fork the example repository](https://github.com/kubeshop/testkube-flux/fork) and clone it locally.
```bash
git clone https://github.com/$GITHUB_USER/testkube-flux.git
```
**2.** Start a Kubernetes cluster.

You can use Minikube, Kind or any managed cluster with a cloud provider (EKS, GKE, etc). In this example we're using [Kind](https://kind.sigs.k8s.io/).
```bash
kind create cluster
```
**3.** Create a Github Classic Token:
Must be of type [Classic](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#creating-a-personal-access-token-classic) (i.e. starts with ghp_)
```bash
GITHUB_TOKEN=<ghp_>
GITHUB_USER=<username>
```
And export the environment variables in your terminal.

4. Install Flux in the cluster and connect it to the repository
Install the [Flux CLI](https://fluxcd.io/flux/installation/) and run:
```bash
flux bootstrap github \
--owner=$GITHUB_USER \
--repository=testkube-flux \
--path=cluster \
--personal
```
**5.** Create a Flux Source and a Kusktomize Controller.

The following command will create Flux source to tell Flux to apply changes that are created in your repository:
```bash
flux create source git testkube-tests \
--url=https://github.com/$GITHUB_USER/testkube-flux \
--branch=main \
--interval=30s \
--export > ./cluster/flux-system/sources/testkube-tests/test-source.yaml
```
And now create a Flux Kustomize Controller to apply the Testkube Test CRDs in the cluser using [Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/):
```bash
flux create kustomization testkube-test \
--target-namespace=testkube \
--source=testkube-tests \
--path="cluster/testkube" \
--prune=true \
--interval=30s \
--export > ./cluster/flux-system/sources/testkube-tests/testkube-kustomization.yaml
```
**6.** Install Testkube in the cluster.

Install the Testkube CLI from **https://kubeshop.github.io/testkube/installing**.

And run the following command to install Testkube and its components in the cluster:
```bash
testkube install
```
**7.** Create a **Test CRD** with **Testkube** CLI.

In this example the test being used is a Postman test, which you can find in **/img/server/tests/postman-collection.json**.

To create a Kubernetes CRD for the test, run:
```bash
testkube generate tests-crds img/server/tests/postman-collection.json > cluster/testkube/server-postman-test.yaml
```
Note: You can [run Testkube from your CI/CD pipeline](https://docs.testkube.io/integrations/testkube-automation/) in case you want to automate the creation of the Test CRDs.

**8.** Add the generated test to the Kustomize file.

The name of the test file created in the previous step is **server-postman-test.yaml**, add that to the Kustomize file located in [cluster/testkube/kustomization.yaml](https://docs.testkube.io/integrations/testkube-automation/):
```bash
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
+ - server-postman-test.yaml
```
**9.** Push all the changes to your repository.
```bash
git pull origin main
git add -A &amp;&amp; git commit -m "Configure Testkube tests"
git push
```
**10.** Your tests should be applied in the cluster.

To see if Flux detected your changes run:
```bash
flux get kustomizations --watch
```
And to ensure that the test has been created run:
```bash
testkube get test
NAME | TYPE | CREATED | LABELS |
--------------------------+--------------------+-------------------------------+---------------------------------------------------+
postman-collection-test | postman/collection | 2023-01-30 18:04:13 +0000 UTC | kustomize.toolkit.fluxcd.io/name=testkube-test, |
| | | kustomize.toolkit.fluxcd.io/namespace=flux-system |
```
**11.** Run your tests.

Now that you have deployed your tests in a GitOps fashion to the cluster, you can use Testkube to run the tests for you through multiple ways:

- Using the Testkube CLI.
- Using the Testkube Dashboard.
- Running Testkube CLI from a CI/CD pipeline.

We'll use the Testkube CLI for brevity. Run the following command to run the recently created test:

```bash
testkube run test postman-collection-test
```
‍
And see the test result with:

```bash
testkube get execution postman-collection-test-1

Test execution completed with success in 13.345s
```
## GitOps Takeaways

Once fully realized - using GitOps for testing of Kubernetes applications as described above provides a powerful alternative to a more traditional approach where orchestration is tied to your current CI/CD tooling and not closely aligned with the lifecycle of Kubernetes applications.

This tutorial uses Postman collections for testing an API, but you can bring your a whole suite of tests with you to Testkube. Check the documentation for the available test types.

Would love to get your thoughts on the above approach - over-engineering done right? Waste of time? Let us know!

Check Testkube on GitHub — and let us know if you’re missing something we should be adding to make your k8s resource testing easier.

[Download the latest release](https://github.com/kubeshop/testkube/releases) on GitHub.

Check out the [documentation](https://docs.testkube.io/).

Get in touch with us on our [Discord server](https://discord.com/channels/884464549347074049/885185660808474664).



# FAQ
Frequently asked questions regarding your Testkube installation.

### How do I install Testkube?
To install Testkube, you'll need to install Testkube CLI (which is currently available on MacOS, Linux, and Windows) and install Testkube's API in your cluster using Helm.

You can read more about how to install and get Testkube up and running by following the instructions in our [Installation Guide](https://kubeshop.github.io/testkube/installing).

### Can I run any test in Testkube?
Yes, if we're not currently supporting a testing framework you need, you can create your custom executor and configure it to run any type of tests that you want. These custom test types can be added to your Testkube installation and/or contributed to our repo. 

You can read more about creating Custom Executors [here](https://kubeshop.github.io/testkube/test-types/executor-custom#creating-a-custom-executor).

### Can I setup my CI/CD to trigger tests?

There's different ways to integrate Testkube with your CI/CD pipeline. You can directly use the command-line interface, or if you use GitHub, you can create GitHub actions.

Read more about the process [here](https://kubeshop.github.io/testkube/integrations/testkube-automation).

If you're familiar with GitOps, check the [ArgoCD integration guide](https://testkube.kubeshop.io/blog/a-gitops-powered-kubernetes-testing-machine-with-argocd-and-testkube) will be useful, and check or the [Flux integration guide](https://testkube.io/blog/flux-testkube-gitops-testing-is-here).

### **Does Testkube have customer support?**
To contact our team for support, we have a few channels available. 
You can reach us via our [Discord server](https://discord.com/invite/6zupCZFQbe) by simply posting your issues on #testkube-general or #testkube-bugs.

You can also create an issue on [GitHub](https://github.com/kubeshop/testkube).
# Step 2 - Installing the Testkube Agent

Now that you've successfully installed Testkube's CLI, you'll need to install Testkube's agent to initiate a new environment.

To get started, sign into [Testkube](https://cloud.testkube.io) and create an account:

![Sign in to Testkube](../img/sign-in.png)

## Installation Steps

1. After signing in, create your first environment

![Create Environment](../img/create-first-environment.png)

2. Fill in the environment name: 

![Fill in Env Name](../img/fill-in-env-name.png)

3. Copy the Helm install command into your terminal to install the environment and deploy the Testkube agent in your cluster: 

![Copy Helm Command](../img/copy-helm-command.png)

4. Run the command in your terminal and wait for Testkube to detect the connection.

You will need *Helm* installed and `kubectl` configured with access to your Kubernetes cluster: 
- To install `helm` just follow the [install instructions on the Helm web site](https://helm.sh/docs/intro/install/).
- To install `kubectl` follow [Kubernetes docs](https://kubernetes.io/docs/tasks/tools/).

![Install Steps 1](../img/install-steps.png)

5. After some time, you should see the Helm installation notice: 

![Install Steps 2](../img/install-steps-2.png)


## Validating the Installation 

Testkube Cloud will notify if the installation is successful. 

* A green indicator means that your cluster was able to connect to the Testkube Cloud.
* A red indicator indicates that the Testkube Agent can't connect to the Testkube Cloud API (Testkube needs some time to establish a connection, max time is 2-3 minutes).

![Validate Install](../img/validate-install.png)

In case of a RED status you can try to debug the issues with the command below:

```sh 
testkube agent debug
```



By default, Testkube is installed in the `testkube` namespace.
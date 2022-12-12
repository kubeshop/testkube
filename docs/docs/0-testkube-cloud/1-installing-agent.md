---
sidebar_position: 1
sidebar_label: Installing Agent
---
## Istallation steps

1. To add a new agent please create a new environment in: 

![creating-environment](https://user-images.githubusercontent.com/30776/207018935-2e1c9d27-2e84-42f2-89c7-faa81ac6eabd.jpg)


![Pass info](https://user-images.githubusercontent.com/30776/206459262-7e8e5987-f30a-41a5-aada-02a58bfc8b31.png)

2. Fill in environment name: 

![Fill in env name](https://user-images.githubusercontent.com/30776/206459469-ceb3dd3d-0eb5-48ca-89be-6debc807b5d3.png)

3. Copy helm install command into the terminal to install new Testkube environment in Agent mode: 

![Copy helm command](https://user-images.githubusercontent.com/30776/206459486-8c7a50a0-4c7c-43f0-ae6a-5a84941f3613.png)

4. Run the command in your terminal

k
| Keep in mind that you'll need *Helm* installed and `kubectl` configured with access to your Kubernetes cluster: 
| - To install `helm` just follow [install instrcutions on Helm web site](https://helm.sh/docs/intro/install/)
| - To install `kubectl` follow [Kubernetes docs](https://kubernetes.io/docs/tasks/tools/)

![Install steps 1](https://user-images.githubusercontent.com/30776/206460225-a71ee0ef-15f0-482a-a188-f8d0cfc485cb.png)

5. After some time you should see Helm installation notice: 

![Install steps 2](https://user-images.githubusercontent.com/30776/206460312-86211dd2-dc50-48be-b33b-11f07720df0a.png)


## Validating the installation 

Testkube Cloud will notify if the installation status is successful. A green indicator means that you cluster was able to connect to Testkube Cloud.

![Validate install](https://user-images.githubusercontent.com/30776/206461244-f885c270-fc57-4919-9330-89a1ce5ad082.png)

From the other side red indicator means that Testkube Agent can't connect to the Testkube Cloud API.

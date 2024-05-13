# How to Migrate from Testkube Core OSS to Testkube Pro On-Prem

It is possible to deploy Testkube Pro On-Prem within the same k8s cluster where Testkube Core OSS is already running. To achieve this, you should install Testkube Pro On-Prem in a different namespace and connect Testkube Core OSS as an Agent.

:::note
Please note that your test executions will not be migrated to Testkube Pro On-Prem, only Test definitions.
:::

## License

To start with Testkube Pro On-Prem you need to request a license. Depending on your environment requirements it can be either an offline or an online license. Read more about these types of licenses [here](https://docs.testkube.io/testkube-pro-on-prem/articles/usage-guide#license). If you require an online license, it can be acquired [here](https://testkube.io/download). If you need an offline license, please contact us using this [form](https://testkube.io/contact).
There are multiple ways to integrate Testkube Core OSS into your Testkube Pro On-Prem setup. We highly recommend creating a k8s secret, as it provides a more secure way to store sensitive data.

At this point there are two options to deploy Testkube Pro On-Prem:

**Multi-cluster Installation:**

- *Description:* This option enables the connection of multiple Agents from different Kubernetes clusters. It allows you to consolidate all tests in a unified Dashboard, organized by Environments.

- *Requirements:* A domain name and certificates are necessary as the installation exposes Testkube endpoints to the outside world.

- *Benefit:* Offers a comprehensive view across clusters and environments.

**One-cluster Installation:**

- *Description:* With this option, you can connect only one Agent (e.g. your existing Testkube Core OSS) within the same Kubernetes cluster. Access to the Dashboard is achieved through port-forwarding to localhost.

- *Requirements:* No domain names or certificates are required for this approach.

- *Benefit:* Simplified setup suitable for a single-cluster environment without the need for external exposure.

## Multi-cluster Installation

If you decide to go with multiple-cluster installation, please ensure that you have the following prerequisites in place:

- [cert-manager](https://cert-manager.io/docs/installation/) (version 1.14.2+ ) or have your own certificates in place;
- [NGINX Controller](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/) (version 4.8.3+) or any other service of your choice to configure ingress traffic;
- a domain for exposing Tetskube endpoints.

### Ingress

To make a central Testkube Pro On-Prem cluster reachable for multiple Agents we need to expose [endpoints](https://docs.testkube.io/testkube-pro-on-prem/articles/usage-guide#domain) and create certificates.
Testkube Pro On-Prem requires the NGINX Controller and it is the only supported Ingress Controller for now. By default, Testkube Pro On-Prem integrates with cert-manager. However, if you choose to use your own certificates, provide them as specified [here](https://docs.testkube.io/testkube-pro-on-prem/articles/usage-guide#tls).
Create a `values.yaml` with your domain and certificate configuration. Additionally include a secretRef to the secret with the license that was created earlier:

`values.yaml`
```yaml
global:
  domain: you-domain.it.com
  enterpriseLicenseSecretRef: testkube-enterprise-license

  certificateProvider: "cert-manager"
  certManager:
    issuerRef: letsencrypt

```

### Auth

Testkube Pro On-Prem utilizes [Dex](https://dexidp.io/) for authentication and authorization. For detailed instruction on configuring Dex, please refer to the [Identity Provider](https://docs.testkube.io/testkube-pro-on-prem/articles/auth) document. You may start with creating static users if you do not have any Identity Provider. Here is an example of usage:

`values.yaml`
```yaml
dex:
 configTemplate:
   additionalConfig: |
     enablePasswordDB: true
     staticPasswords:
       - email: "admin@example.com"
         # bcrypt hash of the string "password": $(echo password | htpasswd -BinC 10 admin | cut -d: -f2)
         hash: "$2a$10$2b2cU8CPhOTaGrs1HRQuAueS7JTT5ZHsHSzYiFPm1leZck7Mc8T4W"
         username: "admin"
         userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"

```

### Deployment

Now, let’s deploy Testkube Pro On-Prem. Please refer to the installation commands [here](https://docs.testkube.io/testkube-pro-on-prem/articles/usage-guide/#installation). Do not forget to pass your customized `values.yaml` file.

It may take a few minutes for the certificates to be issued and for the pods to reach `Ready` status. Once everything is up and running, you may go to dashboard.your-domain.it.com and log in.

The only thing that is remaining is to connect Testkube Core OSS as an Agent. [Create a new environment](https://docs.testkube.io/testkube-pro/articles/environment-management/#creating-a-new-environment) and duplicate the installation command. Execute this command in the cluster where Testkube Core OSS is deployed to seamlessly upgrade the existing installation to Agent mode. Pay attention to the namespace name, ensuring it aligns with the namespace of Testkube Core OSS.

After running the command, navigate to the Dashboard and you will see all your tests available.

## One-cluster Installation

It is possible to deploy Testkube Pro On-Prem and connect an Agent to it in the same k8s cluster without exposing endpoints to the outside world. You can find all the instructions at [the Testkube Quickstart](../../articles/install/quickstart-install.mdx).
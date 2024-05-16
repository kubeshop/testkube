# Testkube Azure DevOps Pipelines

Testkube's integration with Azure DevOps streamlines the installation of Testkube, enabling the execution of any [Testkube CLI](https://docs.testkube.io/cli/testkube) command within Azure DevOps pipelines. This integration can be effortlessly integrated into your Azure DevOps setup, enhancing your continuous integration and delivery processes.
The Azure DevOps integration offers a versatile solution for managing your pipeline workflows and is compatible with Testkube Pro, Testkube Pro On-Prem and the open-source Testkube platform. It allows Azure DevOps users to effectively utilize Testkube's capabilities within their CI/CD pipelines, providing a robust and flexible framework for test execution and automation.

### Azure DevOps Extension

Install the Testkube CLI extension using the following url:
[https://marketplace.visualstudio.com/items?itemName=Testkube.testkubecli](https://marketplace.visualstudio.com/items?itemName=Testkube.testkubecli)

#### Troubleshooting
For solutions to common issues, such as the `--git` flags causing timeouts, please refer to our [Troubleshooting article](./azure-troubleshooting.md).

## Testkube Pro

### How to configure Testkube CLI action for Testkube Pro and run a test

To use Azure DevOps Pipelines for [Testkube Pro](https://app.testkube.io/), you need to create an [API token](https://docs.testkube.io/testkube-pro/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If a test is already created, you can run it using the command `testkube run test test-name -f`. However, if you need to create a test in this workflow, you need to add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

You'll need to create a `azure-pipelines.yml`` file. This will include the stages, jobs and tasks necessary to execute the workflow

```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: Test
  jobs:
  - job: RunTestkube
    steps:
      - task: SetupTestkube@1
        inputs:
          organization: '$(TK_ORG_ID)'
          environment: '$(TK_ENV_ID)'
          token: '$(TK_API_TOKEN)'
      - script: testkube run test test-name -f
        displayName: Run Testkube Test
```

## Testkube Core OSS

### How to configure the Testkube CLI action for TK OSS and run a test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster, and pass an optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: Test
  jobs:
  - job: RunTestkube
    steps:
      - task: SetupTestkube@1
        inputs:
          namespace: 'custom-testkube-namespace'
          url: 'custom-testkube-url'
      - script: testkube run test test-name -f
        displayName: 'Run Testkube Test'
```

The steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider for how to connect to the Kubernetes cluster from Azure DevOps Pipelines

### How to configure Testkube CLI action for TK OSS and run a test

This workflow establishes a connection to the EKS cluster and creates and runs a test using TK CLI. In this example we also use variables not
 to reveal sensitive data. Please make sure that the following points are satisfied:
- The **AWS_ACCESS_KEY_ID**, **AWS_SECRET_ACCESS_KEY** secrets should contain your AWS IAM keys with proper permissions to connect to the EKS cluster.
- The **AWS_REGION** secret should contain the AWS region where EKS is.
- Tke **EKS_CLUSTER_NAME** secret points to the name of the EKS cluster you want to connect.

```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: Test
  jobs:
  - job: SetupAndRunTestkube
    steps:
      - script: |
          # Setting up AWS credentials
          aws configure set aws_access_key_id $(AWS_ACCESS_KEY_ID)
          aws configure set aws_secret_access_key $(AWS_SECRET_ACCESS_KEY)
          aws configure set region $(AWS_REGION)

          # Updating kubeconfig for EKS
          aws eks update-kubeconfig --name $(EKS_CLUSTER_NAME) --region $(AWS_REGION)
        displayName: 'Setup AWS and Testkube'

      - task: SetupTestkube@1
        inputs:
          organization: '$(TK_ORG_ID)'
          environment: '$(TK_ENV_ID)'
          token: '$(TK_API_TOKEN)'

      - script: testkube run test test-name -f
        displayName: Run Testkube Test

```

### How to connect to GKE (Google Kubernetes Engine) cluster and run a test 

This example connects to a k8s cluster in Google Cloud and creates and runs a test using Testkube Azure DevOps pipeline. Please make sure that the following points are satisfied:
- The **GKE Sevice Account** should have already been created in Google Cloud and added to pipeline variables along with **GKE_PROJECT** value.
- The **GKE_CLUSTER_NAME** and **GKE_ZONE** can be added as environmental variables in the workflow.

```yaml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: SetupGKE
  jobs:
  - job: SetupGCloudAndKubectl
    steps:
    - task: DownloadSecureFile@1
      name: gkeServiceAccount
      inputs:
        secureFile: 'gke-service-account-key.json'
    - task: GoogleCloudSdkInstaller@0
      inputs:
        version: 'latest'
    - script: |
        gcloud auth activate-service-account --key-file $(gkeServiceAccount.secureFilePath)
        gcloud config set project $(GKE_PROJECT)
        gcloud container clusters get-credentials $(GKE_CLUSTER_NAME) --zone $(GKE_ZONE)
      displayName: 'Setup GKE'

- stage: Test
  dependsOn: SetupGKE
  jobs:
  - job: RunTestkube
    steps:
    - task: SetupTestkube@1
      inputs:
        organization: '$(TK_ORG_ID)'
        environment: '$(TK_ENV_ID)'
        token: '$(TK_API_TOKEN)'
    - script: |
        testkube run test test-name -f
      displayName: 'Run Testkube Test'
```

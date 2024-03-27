# Testkube GitHub Action

The Testkube GitHub Action installs Testkube and enables running any [Testkube CLI](https://docs.testkube.io/cli/testkube) command in a GitHub workflow. It is available on Github Marketplace <https://github.com/marketplace/actions/testkube-action>.
The action provides a flexible way to work with your pipeline and can be used with Testkube Pro, Testkube Pro On-Prem and the open source Testkube platform.

## Testkube Pro

### How to configure Testkube CLI action for Testkube Pro and Run a Test

To use this GitHub Action for the [Testkube Pro](https://app.testkube.io/), you need to create an [API token](https://docs.testkube.io/testkube-pro/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If a test is already created, you may directly run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
steps:
  - uses: kubeshop/setup-testkube@v1
    with:
      organization: tkcorg_0123456789abcdef
      environment: tkcenv_fedcba9876543210
      token: tkcapi_0123456789abcdef0123456789abcd

  - run: |
      testkube run test test-name -f 

```
It is recommended that sensitive values should never be stored as plaintext in workflow files, but rather as [secrets](https://docs.github.com/en/actions/security-guides/using-secrets-in-github-actions).  Secrets can be configured at the organization, repository, or environment level, and allow you to store sensitive information in GitHub.

```yaml
steps:
  - uses: kubeshop/setup-testkube@v1
    with:
      organization: ${{ secrets.TkOrganization }}
      environment: ${{ secrets.TkEnvironment }}
      token: ${{ secrets.TkToken }}

  - run: |
      testkube run test test-name -f 

 ```
## Testkube Core OSS

### How to Configure Testkube CLI Actions for Testkube Core OSS and Run a Test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster, and simply passing optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If a test is already created, you may run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
steps: 
  - uses: kubeshop/setup-testkube@v1
    with:
      namespace: custom-testkube

  - run: |
      testkube run test test-name -f 

```

Steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider on how to connect to the Kubernetes cluster from GitHub Action, or check examples in this documentation for selected providers.

### How to Configure Testkube CLI Actions for Testkube Core OSS and Run a Test

This workflow establishes a connection to EKS cluster and creates and runs a test using Testkube CLI. In this example, we also use GitHub secrets not to reveal sensitive data. Please make sure that the following points are satisfied:
- The **_AwsAccessKeyId_**, **_AwsSecretAccessKeyId_** secrets should contain your AWS IAM keys with proper permissions to connect to EKS cluster.
- The **_AwsRegion_** secret should contain an AWS region where EKS is.
- Tke **EksClusterName** secret points to the name of EKS cluster you want to connect.

```yaml
steps:   
  - name: Checkout
    uses: actions/checkout@v4

  - uses: aws-actions/configure-aws-credentials@v4
    with:
      aws-access-key-id: ${{ secrets.aws-access-key }}
      aws-secret-access-key: ${{ secrets.aws-secret-access-key }} 
      aws-region:  ${{ secrets.aws-region }}  

  - run: |
      aws eks update-kubeconfig --name ${{ secrets.eks-cluster-name }} --region ${{ secrets.aws-region }} 

  - uses: kubeshop/setup-testkube@v1
    - 
  - run: |
      testkube run test test-name -f 
```

### How to Connect to GKE (Google Kubernetes Engine) Cluster and Run a Test 

This example connects to a k8s cluster in Google Cloud, creates and runs a test using Testkube GH Action. Please make sure that the following points are satisfied:
- The **_GKE Sevice Account_** should be created prior in Google Cloud and added to GH Secrets along with **_GKE Project_** value;
- The **_GKE Cluster Name_** and **_GKE Zone_** can be added as [environmental variables](https://docs.github.com/en/actions/learn-github-actions/variables) in the workflow.

```yaml
steps:    
  - name: Checkout
    uses: actions/checkout@v4

  - uses: google-github-actions/setup-gcloud@1bee7de035d65ec5da40a31f8589e240eba8fde5
    with:
      service_account_key: ${{ secrets.GKE_SA_KEY }}
      project_id: ${{ secrets.GKE_PROJECT }}

  - run: |-
      gcloud --quiet auth configure-docker

  - uses: google-github-actions/get-gke-credentials@db150f2cc60d1716e61922b832eae71d2a45938f
    with:
      cluster_name: ${{ env.GKE_CLUSTER_NAME }}
      location: ${{ env.GKE_ZONE }}
      credentials: ${{ secrets.GKE_SA_KEY }}

  - uses: kubeshop/setup-testkube@v1
  - run: |
      testkube run test test-name -f 
```
Please consult the official documentation from GitHub on how to connect to GKE for more information [here](https://docs.github.com/en/actions/deployment/deploying-to-google-kubernetes-engine).

### Complete Example of Working GitHub Actions Workflow and Testkube Tests Usage
To integrate Testkube Github Actions into your workflow, please take a look at the example that sets up connection to GKE and creates and runs a test:

```yaml
name: Running Testkube Tests.
on:
  push:
    branches:
      - main
env:
  PROJECT_ID: ${{ secrets.GKE_PROJECT }}
  GKE_CLUSTER: cluster-1    # Add your cluster name here.
  GKE_ZONE: us-central1-c   # Add your cluster zone here.

jobs:
  setup-build-publish-deploy:
    name: Connect to GKE
    runs-on: ubuntu-latest

    steps:
    - name: Checkout
      uses: actions/checkout@v4

    - uses: google-github-actions/setup-gcloud@1bee7de035d65ec5da40a31f8589e240eba8fde5
      with:
        service_account_key: ${{ secrets.GKE_SA_KEY }}
        project_id: ${{ secrets.GKE_PROJECT }}

    - run: |-
        gcloud --quiet auth configure-docker

    - uses: google-github-actions/get-gke-credentials@db150f2cc60d1716e61922b832eae71d2a45938f
      with:
        cluster_name: ${{ env.GKE_CLUSTER }}
        location: ${{ env.GKE_ZONE }}
        credentials: ${{ secrets.GKE_SA_KEY }}


      - uses: kubeshop/setup-testkube@v1
      - run: |
          testkube run test test-name -f 
```


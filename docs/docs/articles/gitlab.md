# Testkube Gitlab CI

The Testkube GitLab CI/CD integration facilitates the installation of Testkube and allows the execution of any [Testkube CLI](https://docs.testkube.io/cli/testkube) command within a GitLab CI/CD pipeline. This integration can be seamlessly incorporated into your GitLab repositories to enhance your CI/CD workflows.
The integration offers a versatile approach to align with your pipeline requirements and is compatible with Testkube Pro, Testkube Pro On-Prem, and the open-source Testkube platform. It enables GitLab users to leverage the powerful features of Testkube directly within their CI/CD pipelines, ensuring efficient and flexible test execution.

## Testkube Pro

### How to configure Testkube CLI action for Testkube Pro and run a test

To use this Gitlab CI for [Testkube Pro](https://app.testkube.io/), you need to create an [API token](https://docs.testkube.io/testkube-pro/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
stages:
  - setup

variables:
  TESTKUBE_API_KEY: tkcapi_0123456789abcdef0123456789abcd
  TESTKUBE_ORG_ID: tkcorg_0123456789abcdef
  TESTKUBE_ENV_ID: tkcenv_fedcba9876543210

setup-testkube:
  stage: setup
  image: 
    name: kubeshop/testkube-cli
    entrypoint: ["/bin/sh", "-c"]
  script:
    - testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID
    - testkube run test test-name -f
```

It is recommended that sensitive values should never be stored as plaintext in workflow files, but rather as [variables](https://docs.gitlab.com/ee/ci/variables/).  Secrets can be configured at the organization, repository, or environment level, and allow you to store sensitive information in Gitlab.

```yaml
stages:
  - setup

setup-testkube:
  stage: setup
  image: 
    name: kubeshop/testkube-cli
    entrypoint: ["/bin/sh", "-c"]
  script:
    - testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID
    - testkube run test test-name -f
 ```
## Testkube Core OSS

### How to configure Testkube CLI action for TK OSS and run a test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster, and pass an optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

In order to connecting to your own cluster, you can put the your kubeconfig file into gitlab variable named KUBECONFIGFILE

```yaml
stages:
  - setup

variables:
  NAMESPACE: custom-testkube

setup-testkube:
  stage: setup
  image: 
    name: kubeshop/testkube-cli
    entrypoint: ["/bin/sh", "-c"]
  script:
    - echo $KUBECONFIGFILE > /tmp/kubeconfig/config
    - export KUBECONFIG=/tmp/kubeconfig/config
    - testkube set context --kubeconfig --namespace $NAMESPACE
    - testkube run test test-name -f
```

The steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider for how to connect to the Kubernetes cluster from Gitlab CI.

### How to configure Testkube CLI action for TK OSS and run a test

This workflow establishes a connection to EKS cluster and creates and runs a test using TK CLI. In this example we also use gitlab variables not to reveal sensitive data. Please make sure that the following points are satisfied:
- The **_AwsAccessKeyId_**, **_AwsSecretAccessKeyId_** secrets should contain your AWS IAM keys with proper permissions to connect to EKS cluster.
- The **_AwsRegion_** secret should contain AWS region where EKS is
- Tke **EksClusterName** secret points to the name of EKS cluster you want to connect.

```yaml
stages:
  - setup
  - test

variables:
  NAMESPACE: custom-testkube

setup-aws:
  stage: setup
  image: 
    name: amazon/aws-cli
    entrypoint: ["/bin/sh", "-c"]
  script:
    - mkdir -p $CI_PROJECT_DIR/tmp/kubeconfig
    - aws configure set aws_access_key_id $AWS_ACCESS_KEY_ID
    - aws configure set aws_secret_access_key $AWS_SECRET_ACCESS_KEY
    - aws configure set region $AWS_REGION
    - aws eks update-kubeconfig --name $EKS_CLUSTER_NAME --region $AWS_REGION --kubeconfig $CI_PROJECT_DIR/tmp/kubeconfig/config
  artifacts:
    paths:
      - $CI_PROJECT_DIR/tmp/kubeconfig
    expire_in: 1 hour

run-testkube:
  stage: test
  image: 
    name: kubeshop/testkube-cli
    entrypoint: ["/bin/sh", "-c"]
  script:
    - export KUBECONFIG=$CI_PROJECT_DIR/tmp/kubeconfig/config
    - testkube set context --kubeconfig --namespace $NAMESPACE
    - echo "Running Testkube test..."
    - testkube run test test-name -f
  dependencies:
    - setup-aws

```
### How to connect to GKE (Google Kubernetes Engine) cluster and run a test 

This example connects to a k8s cluster in Google Cloud, creates and runs a test using Testkube Gitlab CI. Please make sure that the following points are satisfied:
- The **_GKE Sevice Account_** should be created prior in Google Cloud and added to Gitlab CI variables along with **_GKE Project_** value;
- The **_GKE Cluster Name_** and **_GKE Zone_** can be added as [environmental variables](https://docs.gitlab.com/ee/ci/variables/) in the workflow.

```yaml
stages:
  - setup
  - test

variables:
  NAMESPACE: custom-testkube

setup-gcp:
  stage: setup
  image: google/cloud-sdk:latest
  script:
    - mkdir -p $CI_PROJECT_DIR/tmp/kubeconfig
    - export KUBECONFIG=$CI_PROJECT_DIR/tmp/kubeconfig/config
    - echo $GKE_SA_KEY | base64 -d > gke-sa-key.json
    - gcloud auth activate-service-account --key-file=gke-sa-key.json
    - gcloud config set project $GKE_PROJECT
    - gcloud --quiet auth configure-docker
    - gcloud container clusters get-credentials $GKE_CLUSTER_NAME --zone $GKE_ZONE
  artifacts:
    paths:
      - $CI_PROJECT_DIR/tmp/kubeconfig
    expire_in: 1 hour

run-testkube:
  stage: test
  image: 
    name: kubeshop/testkube-cli
    entrypoint: ["/bin/sh", "-c"]
  script:
    - export KUBECONFIG=$CI_PROJECT_DIR/tmp/kubeconfig/config
    - testkube set context --kubeconfig --namespace $NAMESPACE
    - echo "Running Testkube test..."
    - testkube run test test-name -f
  dependencies:
    - setup-gcp
```

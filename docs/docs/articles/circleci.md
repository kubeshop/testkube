# Testkube CircleCI

The Testkube CircleCI integration facilitates the installation of Testkube and allows the execution of any [Testkube CLI](https://docs.testkube.io/cli/testkube) command within a CircleCI pipeline. This integration can be seamlessly incorporated into your CircleCI repositories to enhance your CI/CD workflows.
The integration offers a versatile approach to align with your pipeline requirements and is compatible with Testkube Pro, Testkube Pro On-Prem, and the open-source Testkube platform. It enables CircleCI users to leverage the powerful features of Testkube directly within their CI/CD pipelines, ensuring efficient and flexible test execution.

## Testkube Pro

### How to configure Testkube CLI action for Testkube Pro and run a test

To use CircleCI for [Testkube Pro](https://app.testkube.io/), you need to create an [API token](https://docs.testkube.io/testkube-pro/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
version: 2.1

jobs:
  run-tests:
    docker:
      - image: kubeshop/testkube-cli
    working_directory: /.testkube
    environment:
      TESTKUBE_API_KEY: tkcapi_0123456789abcdef0123456789abcd
      TESTKUBE_ORG_ID: tkcorg_0123456789abcdef
      TESTKUBE_ENV_ID: tkcenv_fedcba9876543210
    steps:
      - run:
          name: "Set Testkube Context"
          command: "testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID --cloud-root-domain testkube.dev"
      - run:
          name: "Trigger testkube test"
          command: "testkube run test test-name -f"

workflows:
  run-tests-workflow:
    jobs:
      - run-tests
```

It is recommended that sensitive values should never be stored as plaintext in workflow files, but rather as [project variables](https://circleci.com/docs/set-environment-variable/#set-an-environment-variable-in-a-project).  Secrets can be configured at the organization or project level and allow you to store sensitive information in CircleCI.

```yaml
version: 2.1

jobs:
  run-tests:
    docker:
      - image: kubeshop/testkube-cli
    working_directory: /.testkube
    steps:
      - run:
          name: "Set Testkube Context"
          command: "testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID --cloud-root-domain testkube.dev"
      - run:
          name: "Trigger testkube test"
          command: "testkube run test test-name -f"

workflows:
  run-tests-workflow:
    jobs:
      - run-tests
```
## Testkube Core OSS

### How to configure Testkube CLI action for TK OSS and run a test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster and pass an optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

In order to connect to your own cluster, you can put your kubeconfig file into CircleCI variable named KUBECONFIGFILE.

```yaml
version: 2.1

jobs:
  run-tests:
    docker:
      - image: kubeshop/testkube-cli
    working_directory: /.testkube
    steps:
      - run: 
          name: "Export kubeconfig"
          command: |
            echo $KUBECONFIGFILE > /.testkube/tmp/kubeconfig/config
            export KUBECONFIG=/.testkube/tmp/kubeconfig/config
      - run:
          name: "Set Testkube Context"
          command: "testkube set context --api-key $TESTKUBE_API_KEY --org $TESTKUBE_ORG_ID --env $TESTKUBE_ENV_ID --cloud-root-domain testkube.dev"
      - run:
          name: "Trigger testkube test"
          command: "testkube run test test-name -f"

workflows:
  run-tests-workflow:
    jobs:
      - run-tests
```

The steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider for how to connect to the Kubernetes cluster from CircleCI.

### How to configure Testkube CLI action for TK OSS and run a test

This workflow establishes a connection to the EKS cluster and creates and runs a test using TK CLI. In this example we also use CircleCI variables not to reveal sensitive data. Please make sure that the following points are satisfied:
- The **_AwsAccessKeyId_**, **_AwsSecretAccessKeyId_** secrets should contain your AWS IAM keys with proper permissions to connect to EKS cluster.
- The **_AwsRegion_** secret should contain the AWS region where EKS is.
- Tke **EksClusterName** secret points to the name of the EKS cluster you want to connect.

```yaml
version: 2.1

jobs:
  setup-aws:
    docker:
      - image: amazon/aws-cli
    steps:
      - run:
          name: "Configure AWS CLI"
          command: |
            mkdir -p /.testkube/tmp/kubeconfig/config
            aws configure set aws_access_key_id $AWS_ACCESS_KEY_ID
            aws configure set aws_secret_access_key $AWS_SECRET_ACCESS_KEY
            aws configure set region $AWS_REGION
            aws eks update-kubeconfig --name $EKS_CLUSTER_NAME --region $AWS_REGION --kubeconfig /.testkube/tmp/kubeconfig/config

  run-testkube-on-aws:
    docker:
      - image: kubeshop/testkube-cli
    working_directory: /.testkube
    environment:
        NAMESPACE: custom-testkube
    steps:
      - run:
          name: "Run Testkube Test on EKS"
          command: |
            export KUBECONFIG=/.testkube/tmp/kubeconfig/config
            testkube set context --kubeconfig --namespace $NAMESPACE
            echo "Running Testkube test..."
            testkube run test test-name -f

workflows:
  aws-testkube-workflow:
    jobs:
      - setup-aws
      - run-testkube-on-aws:
          requires:
            - setup-aws
```

### How to connect to GKE (Google Kubernetes Engine) cluster and run a test 

This example connects to a k8s cluster in Google Cloud then creates and runs a test using Testkube CircleCI. Please make sure that the following points are satisfied:
- The **_GKE Sevice Account_** should already be created in Google Cloud and added to CircleCI variables along with **_GKE Project_** value.
- The **_GKE Cluster Name_** and **_GKE Zone_** can be added as environment variables in the workflow.


```yaml
version: 2.1

jobs:
  setup-gcp:
    docker:
      - image: google/cloud-sdk:latest
    working_directory: /.testkube
    steps:
      - run:
          name: "Setup GCP"
          command: |
            mkdir -p /.testkube/tmp/kubeconfig/config
            export KUBECONFIG=$CI_PROJECT_DIR/tmp/kubeconfig/config
            echo $GKE_SA_KEY | base64 -d > gke-sa-key.json
            gcloud auth activate-service-account --key-file=gke-sa-key.json
            gcloud config set project $GKE_PROJECT
            gcloud --quiet auth configure-docker
            gcloud container clusters get-credentials $GKE_CLUSTER_NAME --zone $GKE_ZONE

  run-testkube-on-gcp:
    docker:
      - image: kubeshop/testkube-cli
    working_directory: /.testkube
    steps:
      - run:
          name: "Run Testkube Test on GKE"
          command: |
            export KUBECONFIG=/.testkube/tmp/kubeconfig/config
            testkube set context --kubeconfig --namespace $NAMESPACE
            testkube run test test-name -f

workflows:
  gke-testkube-workflow:
    jobs:
      - setup-gcp
      - run-testkube-on-gcp:
          requires:
            - setup-gcp
```

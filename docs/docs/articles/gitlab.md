# Testkube Gitlab CI

The Testkube GitLab CI/CD integration facilitates the installation of Testkube and allows the execution of any [Testkube CLI](https://docs.testkube.io/cli/testkube) command within a GitLab CI/CD pipeline. This integration can be seamlessly incorporated into your GitLab repositories to enhance your CI/CD workflows.
The integration offers a versatile approach to align with your pipeline requirements and is compatible with Testkube Cloud, Testkube Enterprise, and the open-source Testkube platform. It enables GitLab users to leverage the powerful features of Testkube directly within their CI/CD pipelines, ensuring efficient and flexible test execution.

## Testkube Cloud

### How to configure Testkube CLI action for TK Cloud and run a test

To use this Gitlab CI for the [Testkube Cloud](https://cloud.testkube.io/), you need to create [API token](https://docs.testkube.io/testkube-cloud/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If test is already created, you may directly run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
stages:
  - setup
  - test

variables:
  TESTKUBE_AGENT_TOKEN: tkcapi_0123456789abcdef0123456789abcd
  TESTKUBE_ORG_ID: tkcorg_0123456789abcdef
  TESTKUBE_ENV_ID: tkcenv_fedcba9876543210

setup-testkube:
  stage: setup
  script:
    - echo "Installing Testkube..."
    - curl -sSLf https://get.testkube.io | sh
    - testkube cloud init --agent-token $TESTKUBE_AGENT_TOKEN --org-id $TESTKUBE_ORG_ID --env-id $TESTKUBE_ENV_ID 

run-testkube-test:
  stage: test
  script:
    - echo "Running Testkube test..."
    - testkube run test test-name -f
```

It is recommended that sensitive values should never be stored as plaintext in workflow files, but rather as [variables](https://docs.gitlab.com/ee/ci/variables/).  Secrets can be configured at the organization, repository, or environment level, and allow you to store sensitive information in Gitlab.

```yaml
stages:
  - setup
  - test

setup-testkube:
  stage: setup
  script:
    - echo "Installing Testkube..."
    - curl -sSLf https://get.testkube.io | sh
    - testkube cloud init --agent-token $TESTKUBE_AGENT_TOKEN --org-id $TESTKUBE_ORG_ID --env-id $TESTKUBE_ENV_ID 

run-testkube-test:
  stage: test
  script:
    - echo "Running Testkube test..."
    - testkube run test test-name -f
 ```
## Testkube OSS

### How to configure Testkube CLI action for TK OSS and run a test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster, and simply passing optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If test is already created, you may directly run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```yaml
stages:
  - setup
  - test

variables:
  NAMESPACE: custom-testkube

setup-testkube:
  stage: setup
  script:
    - echo "Installing Testkube..."
    - curl -sSLf https://get.testkube.io | sh
    - testkube cloud init --namespace $NAMESPACE

run-testkube-test:
  stage: test
  script:
    - echo "Running Testkube test..."
    - testkube run test test-name -f
```

Steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider on how to connect to the Kubernetes cluster from Gitlab CI.

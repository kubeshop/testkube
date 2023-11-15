# Testkube Jenkins

The Testkube Jenkins integration streamlines the installation of Testkube, enabling the execution of any [Testkube CLI](https://docs.testkube.io/cli/testkube) command within Jenkins pipelines. This integration can be effortlessly integrated into your Jenkins setup, enhancing your continuous integration and delivery processes.
This Jenkins integration offers a versatile solution for managing your pipeline workflows and is compatible with Testkube Cloud, Testkube Enterprise, and the open-source Testkube platform. It allows Jenkins users to effectively utilize Testkube's capabilities within their CI/CD pipelines, providing a robust and flexible framework for test execution and automation.

## Testkube Cloud

### How to configure Testkube CLI action for TK Cloud and run a test

To use Jenkins CI/CD for [Testkube Cloud](https://cloud.testkube.io/), you need to create an [API token](https://docs.testkube.io/testkube-cloud/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```groovy
pipeline {
    agent any

    stages {
        stage('Setup Testkube') {
            steps {
                script {
                    // Retrieve credentials
                    def agentToken = credentials('TESTKUBE_AGENT_TOKEN')
                    def orgId = credentials('TESTKUBE_ORG_ID')
                    def envId = credentials('TESTKUBE_ENV_ID')

                    // Install Testkube
                    sh 'curl -sSLf https://get.testkube.io | sh'

                    // Initialize Testkube
                    sh "testkube cloud init --agent-token ${agentToken} --org-id ${orgId} --env-id ${envId} 
                }
            }
        }

        stage('Run Testkube Test') {
            steps {
                // Run a Testkube test
                sh 'testkube run test test-name -f'
            }
        }
    }
}

```

## Testkube OSS

### How to configure Testkube CLI action for TK OSS and run a test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster, and simply passing optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If test is already created, you may directly run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

```groovy
pipeline {
    agent any

    stages {
        stage('Setup Testkube') {
            steps {
                script {
                    // Retrieve credentials
                    def namespace='custom-testkube'

                    // Install Testkube
                    sh 'curl -sSLf https://get.testkube.io | sh'

                    // Initialize Testkube
                    sh "testkube cloud init --namespace ${namespace}"
                }
            }
        }

        stage('Run Testkube Test') {
            steps {
                // Run a Testkube test
                sh 'testkube run test test-name -f'
            }
        }
    }
}
```

Steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider on how to connect to the Kubernetes cluster from Jenkins CI/CD

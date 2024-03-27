# Testkube Jenkins Pipelines

The Testkube Jenkins integration streamlines the installation of Testkube, enabling the execution of any [Testkube CLI](https://docs.testkube.io/cli/testkube) command within Jenkins pipelines. This integration can be effortlessly integrated into your Jenkins setup, enhancing your continuous integration and delivery processes.
This Jenkins integration offers a versatile solution for managing your pipeline workflows and is compatible with Testkube Pro, Testkube Pro On-Prem, and the open-source Testkube platform. It allows Jenkins users to effectively utilize Testkube's capabilities within their CI/CD pipelines, providing a robust and flexible framework for test execution and automation.

### Testkube CLI Jenkins Plugin

Install the Testkube CLI plugin by searching it in the "Available Plugins" section on Jenkins Plugins, or using the following url:
[https://plugins.jenkins.io/testkube-cli](https://plugins.jenkins.io/testkube-cli)

## Testkube Pro

### How to configure Testkube CLI action for Testkube Pro and run a test

To use Jenkins CI/CD for [Testkube Pro](https://app.testkube.io/), you need to create an [API token](https://docs.testkube.io/testkube-pro/articles/organization-management/#api-tokens).
Then, pass the **organization** and **environment** IDs, along with the **token** and other parameters specific for your use case.

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

You'll need to create a Jenkinsfile. This Jenkinsfile should define the stages and steps necessary to execute the workflow

```groovy
pipeline {
    agent any

    environment {
        TK_ORG = credentials("TK_ORG")
        TK_ENV = credentials("TK_ENV")
        TK_API_TOKEN = credentials("TK_API_TOKEN")
    }
    stages {
        stage('Example') {
            steps {
                script {
                    // Setup the Testkube CLI
                    setupTestkube()
                    // Run testkube commands
                    sh 'testkube run test your-test'
                    sh 'testkube run testsuite your-test-suite --some-arg --other-arg'
                }
            }
        }
    }
}
```

## Testkube Core OSS

### How to configure Testkube CLI action for TK OSS and run a test

To connect to the self-hosted instance, you need to have **kubectl** configured for accessing your Kubernetes cluster, and pass an optional namespace, if Testkube is not deployed in the default **testkube** namespace. 

If a test is already created, you can run it using the command `testkube run test test-name -f` . However, if you need to create a test in this workflow, please add a creation command, e.g.: `testkube create test --name test-name --file path_to_file.json`.

you'll need to create a Jenkinsfile. This Jenkinsfile should define the stages and steps necessary to execute the workflow

```groovy
pipeline {
    agent any

    environment {
        TK_NAMESPACE = 'custom-testkube-namespace'
    }
    stages {
        stage('Example') {
            steps {
                script {
                    setupTestkube()
                    sh 'testkube run test your-test'
                }
            }
        }
    }
}
```

The steps to connect to your Kubernetes cluster differ for each provider. You should check the docs of your Cloud provider for how to connect to the Kubernetes cluster from Jenkins CI/CD

### How to configure Testkube CLI action for TK OSS and run a test

This workflow establishes a connection to EKS cluster and creates and runs a test using TK CLI. In this example we also use variables not
 to reveal sensitive data. Please make sure that the following points are satisfied:
- The **_AwsAccessKeyId_**, **_AwsSecretAccessKeyId_** secrets should contain your AWS IAM keys with proper permissions to connect to EKS cluste
r.
- The **_AwsRegion_** secret should contain AWS region where EKS is
- Tke **EksClusterName** secret points to the name of EKS cluster you want to connect.

you'll need to create a Jenkinsfile. This Jenkinsfile should define the stages and steps necessary to execute the workflow

```groovy
pipeline {
    agent any

    environment {
        TK_ORG = credentials("TK_ORG")
        TK_ENV = credentials("TK_ENV")
        TK_API_TOKEN = credentials("TK_API_TOKEN")
    }
    stages {
        stage('Setup Testkube') {
            steps {
                script {
                    // Setting up AWS credentials
                    withCredentials([[$class: 'AmazonWebServicesCredentialsBinding', credentialsId: 'AwsAccessKeyId']]) {
                        sh 'aws configure set aws_access_key_id $AWS_ACCESS_KEY_ID'
                        sh 'aws configure set aws_secret_access_key $AWS_SECRET_ACCESS_KEY'
                        sh 'aws configure set region $AWS_REGION'
                    }

                    // Updating kubeconfig for EKS
                    withCredentials([string(credentialsId: 'EksClusterName', variable: 'EKS_CLUSTER_NAME')]) {
                        sh 'aws eks update-kubeconfig --name $EKS_CLUSTER_NAME --region $AWS_REGION'
                    }

                    // Installing and configuring Testkube based on env vars
                    setupTestkube()

                    // Running Testkube test
                    sh 'testkube run test test-name -f'
                }
            }
        }
    }
}
```

### How to connect to GKE (Google Kubernetes Engine) cluster and run a test 

This example connects to a k8s cluster in Google Cloud, creates and runs a test using Testkube Jenkins workflow. Please make sure that the following points are satisfied:
- The **_GKE Sevice Account_** should be created prior in Google Cloud and added to Jenkins variables along with **_GKE Project_** value;
- The **_GKE Cluster Name_** and **_GKE Zone_** can be added as environmental variables in the workflow.
you'll need to create a Jenkinsfile. This Jenkinsfile should define the stages and steps necessary to execute the workflow

```groovy
pipeline {
    agent any

    environment {
        TK_ORG = credentials("TK_ORG")
        TK_ENV = credentials("TK_ENV")
        TK_API_TOKEN = credentials("TK_API_TOKEN")
    }
    stages {
        stage('Deploy to GKE') {
            steps {
                script {
                    // Activating GKE service account
                    withCredentials([file(credentialsId: 'GKE_SA_KEY', variable: 'GKE_SA_KEY_FILE')]) {
                        sh 'gcloud auth activate-service-account --key-file=$GKE_SA_KEY_FILE'
                    }

                    // Setting GCP project
                    withCredentials([string(credentialsId: 'GKE_PROJECT', variable: 'GKE_PROJECT')]) {
                        sh 'gcloud config set project $GKE_PROJECT'
                    }

                    // Configure Docker with gcloud as a credential helper
                    sh 'gcloud --quiet auth configure-docker'

                    // Get GKE cluster credentials
                    withCredentials([
                        string(credentialsId: 'GKE_CLUSTER_NAME', variable: 'GKE_CLUSTER_NAME'),
                        string(credentialsId: 'GKE_ZONE', variable: 'GKE_ZONE')
                    ]) {
                        sh 'gcloud container clusters get-credentials $GKE_CLUSTER_NAME --zone $GKE_ZONE'
                    }

                    // Installing and configuring Testkube based on env vars
                    setupTestkube()

                    // Running Testkube test
                    sh 'testkube run test test-name -f'
                }
            }
        }
    }

    post {
        always {
            // Clean up
            sh 'rm -f $GKE_SA_KEY_FILE'
        }
    }
}
```

# GitHub Actions

In order to automate Testkube runs, access to a K8s cluster is needed. For example, a configured environment with the set up context and kubeconfig for communication with the K8s cluster.

Testkube uses your K8s context and access settings in order to interact with the cluster and tests resources, etc.

In the next few sections, we will go through the process of Testkube and Helm (for Testkube's release deploy/upgrade) automations with the usage of GitHub Actions and GKE K8s.

## Testkube GitHub Action

The testkube GitHub Action is available here <https://github.com/marketplace/actions/testkube-action> and it enables running the Testkube CLI commands in a GitHub workflow.

The following example shows how to create a test using the GitHub action.

```yaml
  - uses: kubeshop/setup-testkube@v1
  - run: |
      testkube create test --name some-test-name --file path_to_file.json
      testkube run test some-test-name 
 ```

## Configuring Your GH Actions for Access to GKE

To obtain set up access to a GKE (Google Kubernetes Engine) from GH (GitHub) actions, please visit the official documentation from GH: <https://docs.github.com/en/actions/deployment/deploying-to-google-kubernetes-engine>.

1. Create a Service Account (SA).
2. Save it into GH's **Secrets** of the repository.
3. Run either `Helm` or `Kubectl kubtest` commands against the set up GKE cluster.

## Main GH Action Section Configuration

To deploy Testkube to your k8s cluster with `helm`, run the `Deploy` command below and once the step is completed, you may run a test:

```yaml
- name: Deploy
  run: |-
    helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace

- uses: kubeshop/setup-testkube@v1
- run: |
    testkube run test some-test-name -f
```

## Complete Example of GH Actions Workflow, Testkube Deployment and Test Creation

This workflow is executed when there is a change to `charts/**` directories in `main` branch. It authenticates to GKE cluster, deploys Testkube helm-chart in `testkube` namespace, creates and runs a test. 

```yaml
name: Running Testkube Tests.
on:
  push:
    paths:
      - "charts/**"
    branches:
      - main

env:
  PROJECT_ID: ${{ secrets.GKE_PROJECT }}
  GKE_CLUSTER_NAME: ${{ secrets.GKE_CLUSTER_NAME }} 
  GKE_ZONE: ${{ secrets.GKE_ZONE }} 
  DEPLOYMENT_NAME: testkube 

jobs:
  deploy-to-testkube-dev-gke:
    name: Deploy
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Configure Git
        run: |
          git config user.name "$GITHUB_ACTOR"
          git config user.email "$GITHUB_ACTOR@users.noreply.github.com"

      # Setup gcloud CLI
      - uses: google-github-actions/setup-gcloud@94337306dda8180d967a56932ceb4ddcf01edae7
        with:
          service_account_key: ${{ secrets.GKE_SA_KEY }}
          project_id: ${{ secrets.GKE_PROJECT }}

      # Configure Docker to use the gcloud command-line tool as a credential
      # helper for authentication
      - run: |-
          gcloud --quiet auth configure-docker

      # Get the GKE credentials so we can deploy to the cluster
      - uses: google-github-actions/get-gke-credentials@fb08709ba27618c31c09e014e1d8364b02e5042e
        with:
          cluster_name: ${{ env.GKE_CLUSTER_NAME }}
          location: ${{ env.GKE_ZONE }}
          credentials: ${{ secrets.GKE_SA_KEY }}

      - name: Install Helm
        uses: azure/setup-helm@v3
        with:
          version: v3.10.0
          
      - name: Installing repositories
        run: |
          helm repo add helm-charts https://kubeshop.github.io/helm-charts
          helm repo add bitnami bitnami https://charts.bitnami.com/bitnami

      # Deploy Testkube helm chart on a GKE cluster
      - name: Deploy
        run: |-
          helm upgrade --install --atomic --timeout 180s ${{ env.DEPLOYMENT_NAME }} helm-charts/testkube --namespace testkube --create-namespace
          
    # Run a test
      - uses: kubeshop/setup-testkube@v1
      - run: |
          testkube create test --name some-test-name --file path_to_file.json
          testkube run test some-test-name
```

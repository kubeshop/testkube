# TestKube Automation
In order to automate TestKube runs the main and the only thing which is required is an `access to a needed K8S cluster`. E.G. Configured environment with the set up context and kubeconfig for communication with the K8S clustrer. 

As TestKube uses your K8S context and access settings in order to interact with the cluster and test scripts etc. 

In the next few sections we will go through the process of TestKube and Helm (for TestKube's release deploy/upgrade) automations with the usage of GitHUb Actions and GKE K8S.
## Configuring your GH actions for the access to GKE

To get set up access to a GKE from GH actions please visit official documentation from GH: https://docs.github.com/en/actions/deployment/deploying-to-google-kubernetes-engine

1. Create SA (service account)
2. Save it into GH's secrets of the repository
3. Run either `Helm` or `Kubectl kubtest` comamnds against set up GKE cluster.

## Main GH's action section configuration:

To install on Linux or MacOs run 
```sh
      # Deploy into configured GKE cluster:
      - name: Deploy
        run: |-
          helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace
```

Instead of Helm you can run any other k8s-native command. In our case: `kubectl kubtest...`

## Full example of working GH actions workflow and TestKube scripts usage. Can be easily re-used with minimal modifications upon your needs.

To install on Linux or MacOs run 
```sh
name: Releasing Helm charts.

on:
  push:
    paths:
      - 'charts/**'
    branches:
      - main

env:
  PROJECT_ID: ${{ secrets.GKE_PROJECT }}
  GKE_CLUSTER_NAME_DEV: ${{ secrets.GKE_CLUSTER_NAME_DEV }}    # Add your cluster name here.
  GKE_ZONE_DEV: ${{ secrets.GKE_ZONE_DEV }}   # Add your cluster zone here.
  DEPLOYMENT_NAME: testkube # Add your deployment name here.

jobs:
  deploy-to-testkube-dev-gke:
    name: Deploy
    runs-on: ubuntu-latest
    needs: notify_slack_if_release_succeeds
    steps:
      - name: Checkout
        uses: actions/checkout@v2
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
          cluster_name: ${{ env.GKE_CLUSTER_NAME_DEV }}
          location: ${{ env.GKE_ZONE_DEV }}
          credentials: ${{ secrets.GKE_SA_KEY }}

      # Run TestKube script on a GKE cluster
      - name: Deploy
        run: |-
          kubectl testkube scripts run SCRIPT_NAME
```
Along with the `kubectl`comand you can pass all the standart K8S parameters like `--namespace` etc.


If you wish to automate CI/CD part of TestKube release for example you can use `Helm` blocks as follow:

```sh
...
      - name: Install Helm
        uses: azure/setup-helm@v1
        with:
          version: v3.4.0

      - name: Installing repositories
        run: |
          helm repo add helm-charts https://kubeshop.github.io/helm-charts
          helm repo add bitnami https://charts.bitnami.com/bitnami
      
      # Run Helm delpoy/upgrade of the TestKube release on a GKE cluster
      - name: Deploy
        run: |-
          helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace
...
```
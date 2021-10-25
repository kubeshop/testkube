# Configuring your GH actions for the access to GKE

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
          helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace --values ./charts/testkube/values-demo.yaml
```

Instead of Helm you can run any other k8s-native command. In our case: `kubectl kubtest...`

## Full example of working GH actions workflow with slack notifications and TestKube Helm chart usage. Can be easily re-used with minimal modifications upon your needs.

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

      - name: Install Helm
        uses: azure/setup-helm@v1
        with:
          version: v3.4.0

      - name: Installing repositories
        run: |
          helm repo add helm-charts https://kubeshop.github.io/helm-charts
          helm repo add bitnami https://charts.bitnami.com/bitnami

      # Deploy/Upgrade the TtestKube release to the GKE cluster
      - name: Deploy
        run: |-
          helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace --values ./charts/testkube/values-demo.yaml

  notify_slack_if_deploy_dev_succeeds:
    runs-on: ubuntu-latest
    needs: deploy-to-testkube-dev-gke
    steps:
    - name: Slack Notification if the helm release deployment to DEV GKS succeeded.
      uses: rtCamp/action-slack-notify@v2
      env:
        SLACK_CHANNEL: testkube-logs
        SLACK_COLOR: ${{ needs.deploy-to-testkube-dev-gke.result }} # or a specific color like 'good' or '#ff00ff'
        SLACK_ICON: https://github.com/rtCamp.png?size=48
        SLACK_TITLE: Helm chart release successfully deployed into ${{ secrets.GKE_CLUSTER_NAME_DEV }} GKE :party_blob:!
        SLACK_USERNAME: GitHub
        SLACK_LINK_NAMES: true
        SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK }}
        SLACK_FOOTER: "Kubeshop --> TestKube"

  notify_slack_if_deploy_dev_failed:
    runs-on: ubuntu-latest
    needs: deploy-to-testkube-dev-gke
    if: always() && (needs.deploy-to-testkube-dev-gke.result == 'failure')
    steps:
    - name: Slack Notification if the helm release deployment to DEV GKS failed.
      uses: rtCamp/action-slack-notify@v2
      env:
        SLACK_CHANNEL: testkube-logs
        SLACK_COLOR: ${{ needs.deploy-to-testkube-dev-gke.result }} # or a specific color like 'good' or '#ff00ff'
        SLACK_ICON: https://github.com/rtCamp.png?size=48
        SLACK_TITLE: Helm chart release failed to deploy into ${{ secrets.GKE_CLUSTER_NAME_DEV }} GKE! :boom:!
        SLACK_USERNAME: GitHub
        SLACK_LINK_NAMES: true
        SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK }}
        SLACK_FOOTER: "Kubeshop --> TestKube"
```
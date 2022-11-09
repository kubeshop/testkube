---
sidebar_position: 1
---
# Integrating with CI/CD

**Check out our [blog post](https://kubeshop.io/blog/a-gitops-powered-kubernetes-testing-machine-with-argocd-and-testkube) to follow tutorial steps for our GitOps-friendly Cloud-native test orchestration/execution framework.**

In order to automate Testkube runs, access to a  K8S cluster is needed, for example, a configured environment with the set up context and kubeconfig for communication with the K8S cluster.  
Testkube uses your K8S context and access settings in order to interact with the cluster and tests resources, etc.

In the next few sections, we will go through the process of Testkube and Helm (for Testkube's release deploy/upgrade) automations with the usage of GitHub Actions and GKE K8S.

## **Testkube github action**

The testkube github action is available here <https://github.com/marketplace/actions/testkube-cli> and it makes possible running the Testkube cli commands in a github workflow. 
Following example shows how to create a test using the github action, a more complex example can be found [here](https://github.com/kubeshop/helm-charts/blob/59054b87f83f890f4f62cf966ac63fd7e46de336/.github/workflows/testkube-docker-action.yaml).

```yaml

 # Creating test
- name: Create test
  id: create_test
  uses: kubeshop/testkube-docker-action@v1
  with:
    command: create
    resource: test
    namespace: testkube
    parameters: "--type k6/script --name testkube-github-action"
    stdin: "import http from 'k6/http';\nimport { sleep,check } from 'k6';\n\nexport default function () {\n  const baseURI = `${__ENV.TESTKUBE_HOMEPAGE_URI || 'https://testkube.kubeshop.io'}`\n  check(http.get(`${baseURI}/`), {\n    'check testkube homepage home page': (r) =>\n      r.body.includes('Your Friendly Cloud-Native Testing Framework for Kubernetes'),\n  });\n\n\n  sleep(1);\n}\n"

```

## **Configuring your GH Actions for the Access to GKE**

To obtain set up access to a GKE (Google Kubernetes Engine) from GH (GitHub) actions, please visit the official documentation from GH: <https://docs.github.com/en/actions/deployment/deploying-to-google-kubernetes-engine>

1. Create a Service Account (SA).
2. Save it into GH's **Secrets** of the repository.
3. Run either `Helm` or `Kubectl kubtest` commands against the set up GKE cluster.

## **Main GH Action Section Configuration**

To install on Linux or MacOS, run:

```yaml
      # Deploy into configured GKE cluster:
      - name: Deploy
        run: |-
          helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace
```

In addition to Helm, you can run any other K8s-native command. In our case: `kubectl kubtest...`

## **Complete Example of Working GH Actions Workflow and Testkube Tests Usage** 

Testkube tests can be easily re-used with minimal modifications according to your needs.

To run tests on Linux or MacOS:

```yaml
name: Running Testkube Tests.

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

      # Run Testkube test on a GKE cluster
      - name: Run test
        id: run_test
        uses: kubeshop/testkube-docker-action@v1
        with:
          command: run
          resource: test
          parameters: TEST_NAME
```

Along with the `kubectl` command, you can pass all the standard K8s parameters such as `--namespace`, etc.

If you wish to automate the CI/CD part of Testkube's Helm release, use `Helm` blocks as follow:

```yaml
...
      - name: Install Helm
        uses: azure/setup-helm@v1
        with:
          version: v3.4.0

      - name: Installing repositories
        run: |
          helm repo add helm-charts https://kubeshop.github.io/helm-charts
          helm repo add bitnami https://charts.bitnami.com/bitnami
      
      # Run Helm delpoy/upgrade of the Testkube release on a GKE cluster
      - name: Deploy
        run: |-
          helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace
...
```

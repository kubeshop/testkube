## Testkube GitHub Action

The Testkube GitHub Action installs Testkube and enables running any [Testkube CLI](https://docs.testkube.io/cli/testkube) command in a GitHub workflow. It is available on Github Marketplace <https://github.com/marketplace/actions/testkube-action>.
The action provides a flexible way to work with your pipeline and can be used with Testkube Cloud or self-hosted platform (self-hosted platform means a k8s cluster that uses an open source Testkube solution (OSS), not Cloud).

The following example shows how to create and run a test using the GitHub action on the [Teskube cloud](https://cloud.testkube.io/) instance. Please note that there are no additional steps needed to connect to the k8s cluster as all the necessary data are provided as inputs:

```yaml
steps:
  # Setup Testkube
  - uses: kubeshop/setup-testkube@v1
    with:
      organization: ${{ secrets.TkOrganization }}
      environment: ${{ secrets.TkEnvironment }}
      token: ${{ secrets.TkToken }}

  # Use CLI with a shell script
  - run: |
      testkube create test --name some-test-name --file path_to_file.json
      testkube run test some-test-name -f
```

Create and run tests on self-hosted platform. Please mind that it requires establishing connection with the k8s cluster:
```yaml
steps:
  - uses: aws-actions/configure-aws-credentials@v4
    with:
      aws-access-key-id: ${{ secrets.AwsAccessKeyId }}
      aws-secret-access-key: ${{ secrets.AwsSecretAccessKey }}
      aws-region: ${{ secrets.AwsRegion }}

  - run: |
      aws eks update-kubeconfig --name ${{ secrets.EksClusterName }} --region ${{ secrets.AwsRegion }}

  # Setup Testkube
  - uses: kubeshop/setup-testkube@v1
  # Use CLI with a shell script
  - run: |
      testkube create test --name some-test-name --file path_to_file.json
      testkube run test some-test-name -f 
 ```
:::note
Please note that it is required to have Testkube Helm charts deployed into your k8s cluster prior to creating and executing tests. We advise to do this in a separate workflow.
:::

Example of deploying Testkube Helm charts that will be connected to the Cloud instance:

```yaml
steps:
    
  - name: Installing repositories
    run: |
      helm repo add helm-charts https://kubeshop.github.io/helm-charts
      helm repo add bitnami bitnami https://charts.bitnami.com/bitnami
      
  - name: Deploy
    run: |-
      helm upgrade --install --reuse-values --create-namespace testkube kubeshop/testkube --set testkube-api.cloud.key=${{ secrets.CLOUD_KEY }}  --set testkube-api.cloud.orgId=${{ secrets.CLOUD_ORG }} --set testkube-api.cloud.envId=${{ secrets.ENV_ID }} --set testkube-api.minio.enabled=false --set mongodb.enabled=false --set testkube-dashboard.enabled=false --set testkube-api.cloud.url=agent.testkube.io:443 --namespace testkube

```

Example of deploying Testkube Helm charts using the OSS solution:

```yaml
steps:
  - uses: aws-actions/configure-aws-credentials@v4
    with:
    aws-access-key-id: ${{ secrets.AwsAccessKeyId }}
    aws-secret-access-key: ${{ secrets.AwsSecretAccessKey }}
    aws-region: ${{ secrets.AwsRegion }}

  - run: |
      aws eks update-kubeconfig --name ${{ secrets.EksClusterName }} --region ${{ secrets.AwsRegion }}  
      
  - name: Installing repositories
    run: |
      helm repo add helm-charts https://kubeshop.github.io/helm-charts
      helm repo add bitnami bitnami https://charts.bitnami.com/bitnami
      
  - name: Deploy
    run: |-
    #OSS solution
      helm upgrade --install --atomic --timeout 180s testkube helm-charts/testkube --namespace testkube --create-namespace
```
Please take a look at the [GH workflow](https://github.com/kubeshop/helm-charts/blob/develop/.github/workflows/helm-releaser-testkube-charts.yaml#L146) that is used in Testkube to deploy OSS solution to GKE cluster.

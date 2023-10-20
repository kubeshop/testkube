# Run Tests with GitHub Actions

**[Testkube Action](https://github.com/marketplace/actions/testkube-action)** is a GitHub Action for running tests, test suites and obtain results directly in the GitHub's workflow.
The action provides you with Testkube CLI that enables building pipelines more efficiently. 

## Usage
To use the action in your GitHub workflow, please place the ``kubeshop/setup-testkube@v1`` action into your file. The configuration options are described in the `Inputs` section and may vary depending on the Testkube solution you are using (cloud or self-hosted) and your needs.

### Testkube Cloud
To use this GitHub Action for the [Testkube Cloud](https://cloud.testkube.io/), you need to [create an API token](https://docs.testkube.io/testkube-cloud/articles/organization-management/#api-tokens).

Then, pass the **organization** and **environment** IDs for the test, along with the **token** and other parameters specific for your use case:

```yaml
uses: kubeshop/setup-testkube@v1
with:
  # Instance
  organization: tkcorg_0123456789abcdef
  environment: tkcenv_fedcba9876543210
  token: tkcapi_0123456789abcdef0123456789abcd
  ```

It will probably be unsafe to keep this directly in the workflow's YAML configuration, so you may want to use [GitHub's secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets) instead.

```yaml
uses: kubeshop/setup-testkube@v1
with:
  # Instance
  organization: ${{ secrets.TkOrganization }}
  environment: ${{ secrets.TkEnvironment }}
  token: ${{ secrets.TkToken }}
  ```

### Self-hosted Instance


To run the test on self-hosted instance, you need to have `kubectl` configured for accessing your Kubernetes cluster, and simply passing optional namespace, if the Testkube is not deployed in the default testkube namespace, i.e

```yaml
uses: kubeshop/setup-testkube@v1
with:
  namespace: custom-testkube
  ```

### Examples

Create and run a test on a self-hosted instance:

```yaml
steps:
  - uses: kubeshop/setup-testkube@v1
    with:
      namespace: custom-testkube
  - run: |
      testkube create test --name some-test-name --file path_to_file.json
      testkube run test some-test-name

  ```
Create and run a test on AWS EKS:

```yaml
steps:
  # Set up Kubectl (AWS EKS)
  - uses: aws-actions/configure-aws-credentials@v4
    with:
      aws-access-key-id: ${{ secrets.AwsAccessKeyId }}
      aws-secret-access-key: ${{ secrets.AwsSecretAccessKey }}
      aws-region: ${{ secrets.AwsRegion }}
  - run: |
      aws eks update-kubeconfig --name ${{ secrets.EksClusterName }} --region ${{ secrets.AwsRegion }}

  - uses: kubeshop/setup-testkube@v1
  - run: |
      testkube create test --name some-test-name --file path_to_file.json
      testkube run test some-test-name
```

Create and run a test on the Cloud instance:
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
      testkube run test some-test-name
```


## Inputs
Besides common inputs, there are some different for kubectl and Cloud connection.
### Common

```sh
| Required | Name            | Description
+----------+-----------------+-----------------------------------------------------------
|    ✗     | channel             | Distribution channel to install the latest application from - one of stable or beta (default: stable)
|    ✗     | version             | Static Testkube CLI version to force its installation instead of the latest
```

### Kubernetes (kubectl)

```sh
| Required | Name            | Description
+----------+-----------------+-----------------------------------------------------------
|    ✗     | namespace       | Set namespace where Testkube is deployed to (default: testkube)
```

### Cloud and Enterprise

```sh
| Required | Name	      | Description
+----------+--------------+------------------------------------------------------------------------------------------------------------------------------
|     ✓    | organization |	The organization ID from Testkube Cloud or Enterprise - it starts with tkc_org, you may find it i.e. in the dashboard's URL.
|     ✓	   | environment  | The environment ID from Testkube Cloud or Enterprise - it starts with tkc_env, you may find it i.e. in the dashboard's URL.
|     ✓	   | token        |	API token that has at least a permission to run specific test or test suite. Read more about creating API token in Testkube Cloud or Enterprise.
|     ✗    | url          | URL of the Testkube Enterprise instance, if applicable.
```

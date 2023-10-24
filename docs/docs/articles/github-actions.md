## Testkube GitHub Action

The Testkube GitHub Action installs Testkube and enables running any [Testkube CLI](https://docs.testkube.io/cli/testkube) command in a GitHub workflow. It is available on Github Marketplace <https://github.com/marketplace/actions/testkube-action>.
The action provides a flexible way to work with your pipeline and can be used with Testkube Cloud, Testkube Enterprise, and an open source Testkube platforms.

### How to run a test on TK Cloud on every PR with this GH Action

The following example shows how to create and run a test using the GitHub action on the [Teskube Cloud](https://cloud.testkube.io/) instance on every opened Pull Request. Please note that there are no additional steps needed to connect to the k8s cluster as all the necessary data are provided as inputs. Do not forget to replace `organization`, `environment` and `token` with your own values.

```yaml
name: Run tests on Pull Request
on:
  pull_request_target:
    types:
      - opened
jobs:
  main:
    name: Install Testkube CLI and Run Tests   
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        
      - uses: kubeshop/setup-testkube@v1
        with:
          organization: tk-organization
          environment: tk-environment
          token: tk-token

      - run: |
          testkube create test --name test-name --file path_to_file.json
          testkube run test test-name -f
```
### How to run a test on your self-hosted Testkube instance on every PR with this GH Action

You can use Testkube Github Action with all Cloud Providers. We will use AWS Cloud here as an example. Please do not forget to replace `aws-access-key-id`, `aws-access-key-id`, `aws-region`, `eks-cluster-name` and `aws-region` values with your own.
```yaml
name: Run tests on Pull Request
on:
  pull_request_target:
    types:
      - opened
jobs:
  main:
    name: Install Testkube CLI and Run Tests
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: aws-access-key
          aws-secret-access-key: aws-secret-access-key
          aws-region: aws-region

      - run: |
          aws eks update-kubeconfig --name eks-cluster-name --region aws-region

      - uses: kubeshop/setup-testkube@v1
      - run: |
          testkube create test --name test-name --file path_to_file.json
          testkube run test test-name -f 
 ```
For a different provider (GKE, Azure) the connection to a k8s cluster will differ so please consult with official documentation of every platform in advance.

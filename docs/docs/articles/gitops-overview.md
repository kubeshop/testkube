# GitOps Cloud Native Testing

One of the major trends in contemporary cloud native application development is the adoption of GitOps; managing the state of your Kubernetes cluster(s) in Git - with all the bells and whistles provided by modern Git platforms like GitHub and GitLab in regard to workflows, auditing, security, tooling, etc. Tools like ArgoCD or Flux are used to do the heavy lifting of keeping your Kubernetes cluster in sync with your Git repository; as soon as a difference is detected between Git and your cluster, it is deployed to ensure that your repository is the source-of-truth for your runtime environment.

Testkube is the first GitOps-friendly Cloud-native test orchestration/execution framework to ensure that your QA efforts align with this new approach to application configuration and cluster configuration management. Combined with the GitOps approach described above, Testkube will include your test artifacts and application configuration in the state of your cluster and make Git the source of truth for these test artifacts.

## Benefits of the GitOps Approach

- Since your tests are included in the state of your cluster you are always able to validate that your application components/services work as required.
- Since tests are executed from inside your cluster there is no need to expose services under test externally purely for the purpose of being able to test them.
- Tests in your cluster are always in sync with the external tooling used for authoring.
- Test execution is not strictly tied to CI but can also be triggered manually for ad-hoc validations or via internal triggers (Kubernetes events).
- You can leverage all your existing test automation assets from Postman, Cypress (even for end-to-end testing), or through executor plugins.

Conceptually, this can be illustrated as follows:

![GitOps CLoud Testing](../img/GitOps-cloud-testing.jpeg)
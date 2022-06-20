# ArgoCD Integration

Please check our Github repository for all stuff related to integration between ArgoCD and Testkube
[Testkube ArgoCD](https://github.com/kubeshop/testkube-argocd)

# ArgoCD Rollouts

Testkube supports ArgoCD rollouts by allowing synchronius execution of the tests and/or test suites
via API calls before and/or after ArgoCD rollouts. You can call execution methods with `sync` flag parameter set to `true`
(check Open API spec for details) and analyse execution results (either `failed` or `passed`).

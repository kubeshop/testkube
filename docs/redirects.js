const redirects = [
  // /docs/oldDoc -> /docs/newDoc
  {
    from: "/executor-cypress",
    to: "/test-types/executor-cypress",
  },
  {
    from: "/executor-postman",
    to: "/test-types/executor-postman",
  },
  {
    from: "/executor-soapui",
    to: "/test-types/executor-soapui",
  },
  {
    from: "/executor-k6",
    to: "/test-types/executor-k6",
  },
  {
    from: "/executor-jmeter",
    to: "/test-types/executor-jmeter",
  },
  {
    from: "/executor-kubepug",
    to: "/test-types/executor-kubepug",
  },
  {
    from: "/executor-artillery",
    to: "/test-types/executor-artillery",
  },
  {
    from: "/executor-maven",
    to: "/test-types/executor-maven",
  },
  {
    from: "/executor-gradle",
    to: "/test-types/executor-gradle",
  },
  {
    from: "/executor-ginkgo",
    to: "/test-types/executor-ginkgo",
  },
  {
    from: "/executor-curl",
    to: "/test-types/executor-curl",
  },
  {
    from: "/test-types/executor-custom",
    to: "/test-types/container-executor",
  },
  {
    from: "/UI",
    to: "/articles/testkube-dashboard",
  },
  {
    from: "/tests-running",
    to: "/articles/running-tests",
  },
  {
    from: "/tests-creating",
    to: "/articles/creating-tests",
  },
  {
    from: "/tests-variables",
    to: "/articles/adding-tests-variables",
  },
  {
    from: "/testsuites-running",
    to: "/articles/running-test-suites",
  },
  {
    from: "/testsuites-creating",
    to: "/articles/creating-test-suites",
  },
  {
    from: "/helm-charts",
    to: "/articles/helm-chart",
  },
  {
    from: "/telemetry",
    to: "/articles/telemetry",
  },
  {
    from: "/installing",
    to: "/articles/getting-started-overview",
  },
  {
    from: "/guides/test-suites/testsuites-getting-results",
    to: "/articles/getting-test-suites-results",
  },
  {
    from: "/category/tests",
    to: "/articles/creating-tests",
  },
  {
    from: "/using-testkube/UI",
    to: "/articles/testkube-dashboard",
  },
  {
    from: "/FAQ",
    to: "/articles/common-issues",
  },
  {
    from: "/integrations/testkube-automation",
    to: "/articles/cicd-overview",
  },
  {
    from: "/guides/tests/tests-creating",
    to: "/articles/creating-tests",
  },
  {
    from: "/guides/exposing-testkube/ingress-nginx",
    to: "/articles/exposing-testkube-with-ingress-nginx",
  },
  {
    from: "/guides/exposing-testkube/overview",
    to: "/articles/exposing-testkube",
  },
  {
    from: "/architecture",
    to: "/articles/architecture",
  },
  {
    from: "/integrations/slack-integration",
    to: "/articles/slack-integration",
  },
  {
    from: "/integrations",
    to: "/articles/getting-started-overview",
  },
  {
    from: "/overview/supported-tests",
    to: "/articles/supported-tests",
  },
  {
    from: "/overview/testkube-benefits",
    to: "/articles/testkube-benefits",
  },
  {
    from: "/getting-started/index",
    to: "/articles/getting-started-overview",
  },
  {
    from: "/getting-started/step1-installing-cli",
    to: "/articles/step1-installing-cli",
  },
  {
    from: "/getting-started/step2-installing-cluster-components",
    to: "/articles/step2-installing-cluster-components",
  },
  {
    from: "/getting-started/step3-creating-first-test",
    to: "/articles/step3-creating-first-test",
  },
  {
    from: "/concepts/tests/tests-creating",
    to: "/articles/creating-tests",
  },
  {
    from: "/concepts/tests/tests-running",
    to: "/articles/running-tests",
  },
  {
    from: "/concepts/tests/tests-getting-results",
    to: "/articles/getting-tests-results",
  },
  {
    from: "/concepts/tests/tests-variables",
    to: "/articles/adding-tests-variables",
  },
  {
    from: "/concepts/test-suites/testsuites-creating",
    to: "/articles/creating-test-suites",
  },
  {
    from: "/concepts/test-suites/testsuites-running",
    to: "/articles/running-test-suites",
  },
  {
    from: "/concepts/test-suites/testsuites-getting-results",
    to: "/articles/getting-test-suites-results",
  },
  {
    from: "/concepts/dashboard",
    to: "/articles/testkube-dashboard",
  },
  {
    from: "/concepts/secrets",
    to: "/articles/adding-tests-secrets",
  },
  {
    from: "/concepts/scheduling",
    to: "/articles/scheduling-tests",
  },
  {
    from: "/concepts/artifacts-storage",
    to: "/articles/artifacts-storage",
  },
  {
    from: "/concepts/metrics",
    to: "/articles/metrics",
  },
  {
    from: "/concepts/triggers",
    to: "/articles/test-triggers",
  },
  {
    from: "/concepts/dependencies",
    to: "/articles/testkube-dependencies",
  },
  {
    from: "/concepts/common-issues",
    to: "/articles/common-issues",
  },
  {
    from: "/concepts/test-sources",
    to: "/articles/test-sources",
  },
  {
    from: "/guides/going-to-production/exposing-testkube/overview",
    to: "/articles/exposing-testkube",
  },
  {
    from: "/guides/going-to-production/exposing-testkube/ingress-nginx",
    to: "/articles/exposing-testkube-with-ingress-nginx",
  },
  {
    from: "/guides/going-to-production/authentication/oauth-cli",
    to: "/articles/oauth-cli",
  },
  {
    from: "/guides/going-to-production/authentication/oauth-ui",
    to: "/articles/oauth-dashboard",
  },
  {
    from: "/guides/going-to-production/aws",
    to: "/articles/deploying-in-aws",
  },
  {
    from: "/guides/cicd/index",
    to: "/articles/cicd-overview",
  },
  {
    from: "/guides/cicd/github-actions",
    to: "/articles/github-actions",
  },
  {
    from: "/guides/cicd/gitops/index",
    to: "/articles/gitops-overview",
  },
  {
    from: "/guides/cicd/gitops/flux",
    to: "/articles/flux-integration",
  },
  {
    from: "/guides/cicd/gitops/argocd",
    to: "/articles/argocd-integration",
  },
  {
    from: "/guides/webhooks",
    to: "/articles/webhooks",
  },
  {
    from: "/guides/slack-integration",
    to: "/articles/slack-integration",
  },
  {
    from: "/guides/generate-test-crds",
    to: "/articles/generate-test-crds",
  },
  {
    from: "/guides/logging",
    to: "/articles/logging",
  },
  {
    from: "/guides/uninstall",
    to: "/articles/uninstall",
  },
  {
    from: "/testkube-cloud/intro",
    to: "/testkube-cloud/articles/intro",
  },
  {
    from: "/testkube-cloud/installing-agent",
    to: "/testkube-cloud/articles/installing-agent",
  },
  {
    from: "/testkube-cloud/transition-from-oss",
    to: "/testkube-cloud/articles/transition-from-oss",
  },
  {
    from: "/testkube-cloud/organization-management",
    to: "/testkube-cloud/articles/organization-management",
  },
  {
    from: "/testkube-cloud/environment-management",
    to: "/testkube-cloud/articles/environment-management",
  },
  {
    from: "/testkube-cloud/managing-cli-context",
    to: "/testkube-cloud/articles/managing-cli-context",
  },
  {
    from: "/testkube-cloud/architecture",
    to: "/testkube-cloud/articles/architecture",
  },
  {
    from: "/reference/helm-chart",
    to: "/articles/helm-chart",
  },
  {
    from: "/reference/openapi",
    to: "/openapi",
  },
  {
    from: "/reference/architecture",
    to: "/articles/architecture",
  },
  {
    from: "/reference/telemetry",
    to: "/articles/telemetry",
  },
  {
    from: "/contributing/intro",
    to: "/articles/contributing",
  },
  {
    from: "/contributing/development/index",
    to: "/articles/development",
  },
  {
    from: "/contributing/development/crds",
    to: "/articles/crds",
  },
];

module.exports = redirects;